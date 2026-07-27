import { parseArgs as nodeParseArgs } from "node:util";

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

function colorLog(msg: string, color: string): void {
  // TODO: implement color output
  console.log(msg);
}

function ensureDir(path: string): void {
  // TODO: implement directory creation
}

function confirmOverwrite(filePath: string, force?: boolean): boolean {
  // TODO: implement overwrite confirmation
  return force ?? false;
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
  // TODO: implement image download
  return { success: false, archiveFile: "", repoPath: "", imageName: image };
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
  // TODO: implement download command
  colorLog("download command not yet implemented", "yellow");
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