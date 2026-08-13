import { describe, test, expect, beforeEach, afterEach } from "bun:test";
import { mkdirSync, rmSync, readFileSync, existsSync, writeFileSync } from "fs";
import { join } from "path";
import {
  parseArgs,
  getDefaultDownloadPath,
  parseComposeFile,
  parseExistingUploadScript,
  generateUploadScript,
  uploadScriptSkopeoCmd,
  downloadImage,
} from "./skopeo";

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

// ═══════════════════════════════════════════════════════════════════════════════
// parseArgs tests
// ═══════════════════════════════════════════════════════════════════════════════

describe("parseArgs", () => {
  test("parses simple command", () => {
    const result = parseArgs(["download"]);
    expect(result.command).toBe("download");
    expect(result.positional).toEqual([]);
    expect(result.options).toEqual({});
  });

  test("parses command with positional args", () => {
    const result = parseArgs(["download", "nginx:latest"]);
    expect(result.command).toBe("download");
    expect(result.positional).toEqual(["nginx:latest"]);
    expect(result.options).toEqual({});
  });

  test("parses command with multiple positional args", () => {
    const result = parseArgs(["compose", "docker-compose.yml", "extra"]);
    expect(result.command).toBe("compose");
    expect(result.positional).toEqual(["docker-compose.yml", "extra"]);
  });

  test("parses --key value options", () => {
    const result = parseArgs(["download", "nginx", "--save", "/tmp/images"]);
    expect(result.command).toBe("download");
    expect(result.positional).toEqual(["nginx"]);
    expect(result.options).toEqual({ save: "/tmp/images" });
  });

  test("parses boolean flags", () => {
    const result = parseArgs(["download", "nginx", "--overwrite"]);
    expect(result.command).toBe("download");
    expect(result.positional).toEqual(["nginx"]);
    expect(result.options).toEqual({ overwrite: true });
  });

  test("parses mixed options and flags", () => {
    const result = parseArgs([
      "compose",
      "docker-compose.yml",
      "--save",
      "/tmp",
      "--overwrite",
      "--filter",
      "nginx",
      "--registry",
      "my.registry.com",
    ]);
    expect(result.command).toBe("compose");
    expect(result.positional).toEqual(["docker-compose.yml"]);
    expect(result.options).toEqual({
      save: "/tmp",
      overwrite: true,
      filter: "nginx",
      registry: "my.registry.com",
    });
  });

  test("handles empty args", () => {
    const result = parseArgs([]);
    expect(result.command).toBe("");
    expect(result.positional).toEqual([]);
    expect(result.options).toEqual({});
  });

  test("handles flags before positional args", () => {
    const result = parseArgs(["--save", "/tmp", "download", "nginx"]);
    expect(result.command).toBe("");
    expect(result.positional).toEqual(["download", "nginx"]);
    expect(result.options).toEqual({ save: "/tmp" });
  });

  test("handles multiple boolean flags", () => {
    const result = parseArgs(["download", "nginx", "--overwrite", "--no-upload-script"]);
    expect(result.options.overwrite).toBe(true);
    expect(result.options["no-upload-script"]).toBe(true);
  });
});

// ═══════════════════════════════════════════════════════════════════════════════
// getDefaultDownloadPath tests
// ═══════════════════════════════════════════════════════════════════════════════

describe("getDefaultDownloadPath", () => {
  const originalPlatform = process.platform;
  const originalHome = process.env.HOME;
  const originalUserProfile = process.env.USERPROFILE;

  afterEach(() => {
    // Restore original values
    Object.defineProperty(process, "platform", { value: originalPlatform });
    if (originalHome !== undefined) {
      process.env.HOME = originalHome;
    } else {
      delete process.env.HOME;
    }
    if (originalUserProfile !== undefined) {
      process.env.USERPROFILE = originalUserProfile;
    } else {
      delete process.env.USERPROFILE;
    }
  });

  test("returns HOME/Downloads on Unix", () => {
    Object.defineProperty(process, "platform", { value: "linux" });
    process.env.HOME = "/home/testuser";
    const result = getDefaultDownloadPath();
    expect(result).toBe("/home/testuser/Downloads");
  });

  test("returns USERPROFILE/Downloads on Windows", () => {
    Object.defineProperty(process, "platform", { value: "win32" });
    process.env.USERPROFILE = "C:\\Users\\testuser";
    const result = getDefaultDownloadPath();
    expect(result).toBe("C:\\Users\\testuser\\Downloads");
  });

  test("falls back to HOME on Windows if USERPROFILE is missing", () => {
    Object.defineProperty(process, "platform", { value: "win32" });
    delete process.env.USERPROFILE;
    process.env.HOME = "/home/testuser";
    const result = getDefaultDownloadPath();
    expect(result).toBe("/home/testuser/Downloads");
  });

  test("returns cwd/Downloads if no HOME set", () => {
    Object.defineProperty(process, "platform", { value: "linux" });
    delete process.env.HOME;
    const result = getDefaultDownloadPath();
    expect(result).toMatch(/\/Downloads$/);
  });
});

// ═══════════════════════════════════════════════════════════════════════════════
// parseComposeFile tests
// ═══════════════════════════════════════════════════════════════════════════════

describe("parseComposeFile", () => {
  beforeEach(() => {
    mkdirSync(TEST_DIR, { recursive: true });
  });

  afterEach(() => {
    rmSync(TEST_DIR, { recursive: true, force: true });
  });

  test("parses simple compose file", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  nginx:
    image: nginx:latest
  redis:
    image: redis:alpine
`;
    writeFileSync(composePath, content);

    const result = parseComposeFile(composePath);
    expect(result).toEqual(["nginx:latest", "redis:alpine"]);
  });

  test("deduplicates images", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  web1:
    image: nginx:latest
  web2:
    image: nginx:latest
`;
    writeFileSync(composePath, content);

    const result = parseComposeFile(composePath);
    expect(result).toEqual(["nginx:latest"]);
  });

  test("strips quotes from image names", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  web:
    image: "nginx:latest"
`;
    writeFileSync(composePath, content);

    const result = parseComposeFile(composePath);
    expect(result).toEqual(["nginx:latest"]);
  });

  test("strips single quotes from image names", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  web:
    image: 'nginx:latest'
`;
    writeFileSync(composePath, content);

    const result = parseComposeFile(composePath);
    expect(result).toEqual(["nginx:latest"]);
  });

  test("filters images by keyword", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  nginx:
    image: nginx:latest
  redis:
    image: redis:alpine
  postgres:
    image: postgres:15
`;
    writeFileSync(composePath, content);

    const result = parseComposeFile(composePath, "nginx");
    expect(result).toEqual(["nginx:latest"]);
  });

  test("filter is case-sensitive", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  nginx:
    image: nginx:latest
`;
    writeFileSync(composePath, content);

    const result = parseComposeFile(composePath, "Nginx");
    expect(result).toEqual([]);
  });

  test("returns empty array for empty compose file", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    writeFileSync(composePath, "services: {}");

    const result = parseComposeFile(composePath);
    expect(result).toEqual([]);
  });

  test("handles images with registry prefix", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  app:
    image: docker.io/library/nginx:latest
`;
    writeFileSync(composePath, content);

    const result = parseComposeFile(composePath);
    expect(result).toEqual(["docker.io/library/nginx:latest"]);
  });

  test("throws for non-existent file", () => {
    expect(() => {
      parseComposeFile(join(TEST_DIR, "nonexistent.yml"));
    }).toThrow();
  });
});

// ═══════════════════════════════════════════════════════════════════════════════
// parseExistingUploadScript tests
// ═══════════════════════════════════════════════════════════════════════════════

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
    const content = `# Auto-generated upload script
$images = @(
    "nginx-latest.tar|docker://docker.senjone.com/library/nginx"
);`;
    writeFileSync(scriptPath, content);

    const result = parseExistingUploadScript(scriptPath);
    expect(result).toEqual(["nginx-latest.tar|docker://docker.senjone.com/library/nginx"]);
  });

  test("parses multiple entries", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = `# Auto-generated upload script
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
    const content = `# Auto-generated upload script
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

  test("handles empty script", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    writeFileSync(scriptPath, "");

    const result = parseExistingUploadScript(scriptPath);
    expect(result).toEqual([]);
  });

  test("handles script with no entries", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = `$images = @();`;
    writeFileSync(scriptPath, content);

    const result = parseExistingUploadScript(scriptPath);
    expect(result).toEqual([]);
  });
});

// ═══════════════════════════════════════════════════════════════════════════════
// generateUploadScript tests
// ═══════════════════════════════════════════════════════════════════════════════

describe("generateUploadScript", () => {
  beforeEach(() => {
    mkdirSync(TEST_DIR, { recursive: true });
  });

  afterEach(() => {
    rmSync(TEST_DIR, { recursive: true, force: true });
  });

  test("creates new script when none exists", () => {
    const entries = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries, TEST_DIR, null, "docker.senjone.com");

    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    expect(existsSync(scriptPath)).toBe(true);
    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("nginx-latest.tar");
    expect(content).toContain("docker.senjone.com");
  });

  test("appends new entries to existing script", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");

    // First download
    const entries1 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries1, TEST_DIR, null, "docker.senjone.com");

    // Second download
    const entries2 = [makeResult("ubuntu-latest.tar", "library/ubuntu")];
    generateUploadScript(entries2, TEST_DIR, null, "docker.senjone.com");

    // Should contain both entries
    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("nginx-latest.tar");
    expect(content).toContain("ubuntu-latest.tar");
  });

  test("does not add duplicate entries", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");

    // First download
    const entries1 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries1, TEST_DIR, null, "docker.senjone.com");

    // Second download with same image
    const entries2 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries2, TEST_DIR, null, "docker.senjone.com");

    // Should still have only one entry
    const content = readFileSync(scriptPath, "utf-8");
    const matches = content.match(/nginx-latest\.tar/g);
    expect(matches?.length).toBe(1);
  });

  test("preserves existing entries when adding new ones", () => {
    const scriptPath = join(TEST_DIR, "upload_all.ps1");

    // First download
    const entries1 = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries1, TEST_DIR, null, "docker.senjone.com");

    // Add two more
    const entries2 = [
      makeResult("ubuntu-latest.tar", "library/ubuntu"),
      makeResult("redis-alpine.tar", "library/redis:alpine"),
    ];
    generateUploadScript(entries2, TEST_DIR, null, "docker.senjone.com");

    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("nginx-latest.tar");
    expect(content).toContain("ubuntu-latest.tar");
    expect(content).toContain("redis-alpine.tar");

    // Verify correct count in script
    expect(content).toContain("$images.Count");
  });

  test("skips failed entries", () => {
    const entries = [
      {
        success: false,
        archiveFile: `${TEST_DIR}/failed.tar`,
        repoPath: "library/failed",
        imageName: "docker.io/library/failed",
      },
      makeResult("success.tar", "library/success"),
    ];
    generateUploadScript(entries, TEST_DIR, null, "docker.senjone.com");

    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = readFileSync(scriptPath, "utf-8");
    expect(content).not.toContain("failed.tar");
    expect(content).toContain("success.tar");
  });

  test("does nothing when all entries failed", () => {
    const entries = [
      {
        success: false,
        archiveFile: `${TEST_DIR}/failed.tar`,
        repoPath: "library/failed",
        imageName: "docker.io/library/failed",
      },
    ];
    generateUploadScript(entries, TEST_DIR, null, "docker.senjone.com");

    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    expect(existsSync(scriptPath)).toBe(false);
  });

  test("includes correct registry in upload paths", () => {
    const entries = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries, TEST_DIR, null, "custom.registry.io");

    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("custom.registry.io/library/nginx");
  });

  test("uses custom skopeo path when provided", () => {
    const entries = [makeResult("nginx-latest.tar", "library/nginx")];
    generateUploadScript(entries, TEST_DIR, "/usr/local/bin/skopeo", "docker.senjone.com");

    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    const content = readFileSync(scriptPath, "utf-8");
    expect(content).toContain("/usr/local/bin/skopeo");
  });

  test("references bundled skopeo.exe in the script directory on Windows", () => {
    expect(uploadScriptSkopeoCmd(null, true)).toBe('"$PSScriptRoot\\skopeo.exe"');
    expect(uploadScriptSkopeoCmd("D:\\tools\\skopeo.exe", true)).toBe('"$PSScriptRoot\\skopeo.exe"');
  });

  test("uses custom path or PATH skopeo on non-Windows", () => {
    expect(uploadScriptSkopeoCmd("/usr/local/bin/skopeo", false)).toBe('"/usr/local/bin/skopeo"');
    expect(uploadScriptSkopeoCmd(null, false)).toBe("skopeo");
  });
});

// ═══════════════════════════════════════════════════════════════════════════════
// Integration tests
// ═══════════════════════════════════════════════════════════════════════════════

describe("Integration: parseComposeFile + generateUploadScript", () => {
  beforeEach(() => {
    mkdirSync(TEST_DIR, { recursive: true });
  });

  afterEach(() => {
    rmSync(TEST_DIR, { recursive: true, force: true });
  });

  test("full workflow: parse compose and generate script", () => {
    // Create a compose file
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  web:
    image: nginx:latest
  db:
    image: postgres:15
`;
    writeFileSync(composePath, content);

    // Parse it
    const images = parseComposeFile(composePath);
    expect(images).toEqual(["nginx:latest", "postgres:15"]);

    // Create mock download results
    const results = images.map((img) => ({
      success: true,
      archiveFile: `${TEST_DIR}/${img.replace(/:/g, "-").replace(/\//g, "_")}.tar`,
      repoPath: `library/${img.split(":")[0]}`,
      imageName: `docker.io/library/${img}`,
    }));

    // Generate upload script
    generateUploadScript(results, TEST_DIR, null, "docker.senjone.com");

    // Verify the script
    const scriptPath = join(TEST_DIR, "upload_all.ps1");
    expect(existsSync(scriptPath)).toBe(true);

    const scriptContent = readFileSync(scriptPath, "utf-8");
    expect(scriptContent).toContain("nginx");
    expect(scriptContent).toContain("postgres");
    expect(scriptContent).toContain("docker.senjone.com");
  });

  test("workflow with filter", () => {
    const composePath = join(TEST_DIR, "docker-compose.yml");
    const content = `
services:
  web:
    image: nginx:latest
  db:
    image: postgres:15
  cache:
    image: redis:alpine
`;
    writeFileSync(composePath, content);

    // Only get nginx images
    const images = parseComposeFile(composePath, "nginx");
    expect(images).toEqual(["nginx:latest"]);
  });
});

// ═══════════════════════════════════════════════════════════════════════════════
// Actual download test - requires skopeo >= 1.11 (supports application/vnd.in-toto+json)
// ═══════════════════════════════════════════════════════════════════════════════

describe("downloadImage - actual download (requires skopeo >= 1.11)", () => {
  const DOWNLOAD_DIR = join(TEST_DIR, ".download-test");

  beforeEach(() => {
    mkdirSync(DOWNLOAD_DIR, { recursive: true });
  });

  afterEach(() => {
    rmSync(DOWNLOAD_DIR, { recursive: true, force: true });
  });

  // Skip these tests if skopeo version is too old
  // The application/vnd.in-toto+json MIME type is used by modern Docker Hub images
  // and requires skopeo >= 1.11 to handle correctly
  test.skip("downloads hello-world successfully", async () => {
    const result = await downloadImage("docker.io/library/hello-world:latest", {
      savePath: DOWNLOAD_DIR,
      overwrite: true,
      noUploadScript: true,
      registry: "docker.senjone.com",
    });

    expect(result.success).toBe(true);
    expect(result.repoPath).toBe("library/hello-world");
    expect(result.archiveFile).toContain("hello-world-latest.tar");
    expect(existsSync(result.archiveFile)).toBe(true);

    // Verify file exists and has content
    const { statSync } = await import("fs");
    const stats = statSync(result.archiveFile);
    expect(stats.size).toBeGreaterThan(0);
  }, 30000);

  test("fails gracefully with invalid image", async () => {
    const result = await downloadImage("docker.io/library/nonexistent-image-xyz-123:latest", {
      savePath: DOWNLOAD_DIR,
      overwrite: true,
      noUploadScript: true,
      registry: "docker.senjone.com",
    });

    expect(result.success).toBe(false);
  }, 15000);

  test.skip("skip existing file when overwrite=false", async () => {
    // First download
    const result1 = await downloadImage("docker.io/library/hello-world:latest", {
      savePath: DOWNLOAD_DIR,
      overwrite: true,
      noUploadScript: true,
      registry: "docker.senjone.com",
    });
    expect(result1.success).toBe(true);

    // Second download without overwrite - should skip
    const result2 = await downloadImage("docker.io/library/hello-world:latest", {
      savePath: DOWNLOAD_DIR,
      overwrite: false,
      noUploadScript: true,
      registry: "docker.senjone.com",
    });
    expect(result2.success).toBe(true);
    expect(result2.archiveFile).toBe(result1.archiveFile);
  }, 30000);
});
