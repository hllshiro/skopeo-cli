import { describe, test, expect, beforeEach, afterEach } from "bun:test";
import { mkdirSync, rmSync, readFileSync, existsSync, writeFileSync } from "fs";
import { join } from "path";
import { parseExistingUploadScript, generateUploadScript } from "./skopeo";

const TEST_DIR = join(import.meta.dir, ".test-upload");

// Helper to create a DownloadResult
function makeResult(fileName: string, repoPath: string) {
  return {
    success: true,
    archiveFile: `${TEST_DIR}/${fileName}`,
    repoPath,
    imageName: `docker.io/${repoPath}`,
  };
}

describe("parseExistingUploadScript", () => {
  beforeEach(() => {
    mkdirSync(TEST_DIR, { recursive: true });
  });

  afterEach(() => {
    rmSync(TEST_DIR, { recursive: true, force: true });
  });

  test("returns empty array for non-existent file", () => {
    const scriptPath = join(TEST_DIR, "nonexistent.ps1");
    const result = parseExistingUploadScript(scriptPath);
    expect(result).toEqual([]);
  });

  test("parses single entry", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = `# 自动上传脚本 - 由 skopeo.ts 生成
$images = @(
    "nginx-latest.tar|docker://docker.senjone.com/library/nginx"
);`;
    writeFileSync(scriptPath, content);

    const result = parseExistingUploadScript(scriptPath);
    expect(result).toEqual(["nginx-latest.tar|docker://docker.senjone.com/library/nginx"]);
  });

  test("parses multiple entries", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = `# 自动上传脚本 - 由 skopeo.ts 生成
$images = @(
    "nginx-latest.tar|docker://docker.senjone.com/library/nginx"
    "ubuntu-latest.tar|docker://docker.senjone.com/library/ubuntu"
    "redis-alpine.tar|docker://docker.senjone.com/library/redis:alpine"
);`;
    writeFileSync(scriptPath, content);

    const result = parseExistingUploadScript(scriptPath);
    expect(result).toEqual([
      "nginx-latest.tar|docker://docker.senjone.com/library/nginx",
      "ubuntu-latest.tar|docker://docker.senjone.com/library/ubuntu",
      "redis-alpine.tar|docker://docker.senjone.com/library/redis:alpine",
    ]);
  });

  test("ignores lines outside $images array", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = `# 自动上传脚本 - 由 skopeo.ts 生成
$images = @(
    "nginx-latest.tar|docker://docker.senjone.com/library/nginx"
);
for ($i = 0; $i -lt $images.Count; $i++) {
    $parts = $images[$i] -split '\\|'
}`;
    writeFileSync(scriptPath, content);

    const result = parseExistingUploadScript(scriptPath);
    expect(result).toEqual(["nginx-latest.tar|docker://docker.senjone.com/library/nginx"]);
  });
});

describe("generateUploadScript - append behavior", () => {
  beforeEach(() => {
    mkdirSync(TEST_DIR, { recursive: true });
  });

  afterEach(() => {
    rmSync(TEST_DIR, { recursive: true, force: true });
  });

  test("creates new script when none exists", () => {
    const entries = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries, TEST_DIR, null);

    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    expect(existsSync(scriptPath)).toBe(true);
    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("nginx-latest.tar");
  });

  test("appends new entries to existing script", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");

    // First download
    const entries1 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries1, TEST_DIR, null);

    // Second download
    const entries2 = [makeResult("ubuntu-latest.tar", "library/ubuntu")];
    generateUploadScript(entries2, TEST_DIR, null);

    // Should contain both entries
    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("nginx-latest.tar");
    expect(content).toContain("ubuntu-latest.tar");
  });

  test("does not add duplicate entries", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");

    // First download
    const entries1 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries1, TEST_DIR, null);

    // Second download with same image
    const entries2 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries2, TEST_DIR, null);

    // Should still have only one entry
    const content = readFileSync(scriptPath, "utf-8");
    const matches = content.match(/nginx-latest\.tar/g);
    expect(matches?.length).toBe(1);
  });

  test("preserves existing entries when adding new ones", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");

    // First download
    const entries1 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries1, TEST_DIR, null);

    // Add two more
    const entries2 = [
      makeResult("ubuntu-latest.tar", "library/ubuntu"),
      makeResult("redis-alpine.tar", "library/redis:alpine"),
    ];
    generateUploadScript(entries2, TEST_DIR, null);

    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("nginx-latest.tar");
    expect(content).toContain("ubuntu-latest.tar");
    expect(content).toContain("redis-alpine.tar");

    // Verify correct count in script
    expect(content).toContain("$images.Count");
  });
});
