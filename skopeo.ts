import { existsSync, mkdirSync, unlinkSync } from "fs";

// ── Types ──────────────────────────────────────────────────────────────────

interface DownloadOptions {
  savePath: string;
  platform?: string;
  overwrite: boolean;
  noUploadScript: boolean;
}

interface ComposeOptions {
  savePath: string;
  filter?: string;
  overwrite: boolean;
  noUploadScript: boolean;
}

interface DownloadResult {
  success: boolean;
  archiveFile: string;
  repoPath: string;
  imageName: string;
}

// ── Helpers ────────────────────────────────────────────────────────────────

function parseArgs(
  args: string[]
): { command: string; positional: string[]; options: Record<string, string | boolean> } {
  const positional: string[] = [];
  const options: Record<string, string | boolean> = {};

  let command = "";
  let i = 0;

  // First non-option token is the command
  while (i < args.length && !args[i]!.startsWith("--")) {
    if (!command) {
      command = args[i]!;
    } else {
      positional.push(args[i]!);
    }
    i++;
  }

  // Remaining tokens: --flag or --key value
  while (i < args.length) {
    const arg = args[i]!;
    if (arg.startsWith("--")) {
      const key = arg.slice(2);
      const next = args[i + 1];
      if (next && !next.startsWith("--")) {
        options[key] = next;
        i += 2;
      } else {
        options[key] = true;
        i++;
      }
    } else {
      positional.push(arg);
      i++;
    }
  }

  return { command, positional, options };
}

const COLORS: Record<string, string> = {
  red: "\x1b[31m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  magenta: "\x1b[35m",
  cyan: "\x1b[36m",
  white: "\x1b[37m",
  reset: "\x1b[0m",
};

function colorLog(msg: string, color: string): void {
  const code = COLORS[color.toLowerCase()] || COLORS.white;
  console.log(`${code}${msg}${COLORS.reset}`);
}

function ensureDir(path: string): void {
  mkdirSync(path, { recursive: true });
}

function confirmOverwrite(filePath: string, force?: boolean): boolean {
  if (!existsSync(filePath)) {
    return false;
  }
  if (force) {
    return true;
  }
  colorLog(`文件已存在，跳过（使用 --overwrite 覆盖）: ${filePath}`, "yellow");
  return false;
}

// ── Core logic ─────────────────────────────────────────────────────────────

function parseComposeFile(filePath: string, filter?: string): string[] {
  // TODO: implement compose file parsing
  return [];
}

async function downloadImage(
  image: string,
  opts: DownloadOptions
): Promise<DownloadResult> {
  // 1. Parse image name — strip registry layer
  const parts = image.split("/");
  let repoPath = image;
  if (
    parts.length > 1 &&
    (parts[0]!.includes(".") || parts[0]!.includes(":") || parts[0] === "localhost")
  ) {
    repoPath = parts.slice(1).join("/");
  }

  // 2. Generate filename
  const fileName = repoPath.replace(/:/g, "-").replace(/\//g, "_");
  const archiveFile = `${opts.savePath}/${fileName}.tar`;

  // 3. Check file existence
  if (!confirmOverwrite(archiveFile, opts.overwrite)) {
    if (existsSync(archiveFile)) {
      return { success: true, archiveFile, repoPath, imageName: image };
    }
  }

  // 4. Build skopeo command
  const skopeoArgs = ["copy", "--all"];
  if (opts.platform) {
    const [os, arch] = opts.platform.split("/");
    if (os) skopeoArgs.push("--override-os", os);
    if (arch) skopeoArgs.push("--override-arch", arch);
  }
  skopeoArgs.push(`docker://${image}`, `oci-archive:${archiveFile}`);

  // 5. Execute skopeo
  colorLog(`正在下载镜像: ${image}...`, "green");
  let proc;
  try {
    proc = Bun.spawnSync(["skopeo", ...skopeoArgs]);
  } catch (e: any) {
    colorLog(`skopeo 未安装或不在 PATH 中: ${e.message}`, "red");
    return { success: false, archiveFile, repoPath, imageName: image };
  }

  if (proc.exitCode !== 0) {
    colorLog(`下载镜像失败: ${image}`, "red");
    // Clean up partial file
    if (existsSync(archiveFile)) {
      unlinkSync(archiveFile);
    }
    return { success: false, archiveFile, repoPath, imageName: image };
  }

  colorLog(`已下载: ${image} → ${archiveFile}`, "green");
  return { success: true, archiveFile, repoPath, imageName: image };
}

function generateUploadScript(
  entries: DownloadResult[],
  savePath: string
): void {
  // TODO: implement upload script generation
}

// ── Commands ───────────────────────────────────────────────────────────────

async function composeCommand(
  file: string,
  opts: ComposeOptions
): Promise<void> {
  // TODO: implement compose command
  colorLog("compose command not yet implemented", "yellow");
}

async function downloadCommand(
  image: string,
  opts: DownloadOptions
): Promise<void> {
  // Default save path
  const savePath = opts.savePath || `${process.env.HOME}/Downloads/docker`;
  opts.savePath = savePath;

  // Ensure directory exists
  ensureDir(savePath);

  // Download
  const result = await downloadImage(image, opts);

  if (result.success) {
    colorLog(`已记录: ${image} → docker.senjone.com/${result.repoPath}`, "green");
  }
}

// ── Main entry ─────────────────────────────────────────────────────────────

const args = process.argv.slice(2);
const parsed = parseArgs(args);

switch (parsed.command) {
  case "download": {
    const image = parsed.positional[0];
    if (!image) {
      colorLog("用法: skopeo.ts download <image>", "yellow");
      process.exit(1);
    }
    const opts: DownloadOptions = {
      savePath: (parsed.options.save as string) || "",
      platform: parsed.options.platform as string | undefined,
      overwrite: !!parsed.options.overwrite,
      noUploadScript: !!parsed.options["no-upload-script"],
    };
    await downloadCommand(image, opts);
    break;
  }
  case "compose": {
    const file = parsed.positional[0];
    if (!file) {
      colorLog("用法: skopeo.ts compose <file>", "yellow");
      process.exit(1);
    }
    const opts: ComposeOptions = {
      savePath: (parsed.options.save as string) || "",
      filter: parsed.options.filter as string | undefined,
      overwrite: !!parsed.options.overwrite,
      noUploadScript: !!parsed.options["no-upload-script"],
    };
    await composeCommand(file, opts);
    break;
  }
  default:
    colorLog("用法: skopeo.ts <download|compose> [options]", "yellow");
    colorLog("  download <image>    下载单个 Docker 镜像", "white");
    colorLog("  compose <file>      从 compose 文件批量下载", "white");
    process.exit(1);
}