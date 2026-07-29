import { existsSync, mkdirSync, readFileSync, unlinkSync } from "fs";

// ── Types ──────────────────────────────────────────────────────────────────

interface DownloadOptions {
  savePath: string;
  platform?: string;
  overwrite: boolean;
  noUploadScript: boolean;
  registry: string;
}

interface ComposeOptions {
  savePath: string;
  filter?: string;
  overwrite: boolean;
  noUploadScript: boolean;
  registry: string;
}

interface DownloadResult {
  success: boolean;
  archiveFile: string;
  repoPath: string;
  imageName: string;
}

function getDefaultDownloadPath(): string {
  const platform = process.platform;
  if (platform === "win32") {
    const userProfile = process.env.USERPROFILE;
    if (userProfile) return `${userProfile}\\Downloads`;
  }
  const home = process.env.HOME;
  if (home) return `${home}/Downloads`;
  return `${process.cwd()}/Downloads`;
}

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
  try {
    mkdirSync(path, { recursive: true });
  } catch (e: any) {
    if (e.code !== "EEXIST") throw e;
  }
}

// ── Skopeo discovery ───────────────────────────────────────────────────────

async function findSkopeo(): Promise<string | null> {
  const isWin = process.platform === "win32";
  try {
    const proc = Bun.spawn(
      isWin ? ["where", "skopeo"] : ["which", "skopeo"],
      { stdout: "pipe", stderr: "pipe" }
    );
    const output = await new Response(proc.stdout).text();
    const exitCode = await proc.exited;
    if (exitCode === 0) {
      const firstLine = output.split("\n")[0]?.trim();
      if (firstLine) return firstLine;
    }
  } catch {
    // Command not found
  }
  return null;
}

async function copySkopeoToDir(targetDir: string): Promise<boolean> {
  const skopeoPath = await findSkopeo();
  if (!skopeoPath) return false;

  const isWin = process.platform === "win32";
  const destName = isWin ? "skopeo.exe" : "skopeo";
  const destPath = `${targetDir}/${destName}`;

  if (existsSync(destPath)) return true;

  try {
    if (isWin) {
      Bun.spawn(["cmd", "/c", "copy", skopeoPath.replace(/\//g, "\\"), destPath.replace(/\//g, "\\")]);
    } else {
      Bun.spawn(["cp", skopeoPath, destPath]);
    }
    return true;
  } catch {
    return false;
  }
}

// ── Core logic ─────────────────────────────────────────────────────────────

function parseComposeFile(filePath: string, filter?: string): string[] {
  if (!existsSync(filePath)) {
    colorLog(`Error: file not found ${filePath}`, "red");
    process.exit(1);
  }

  const text = readFileSync(filePath, "utf-8");
  const allImages: string[] = [];
  const imageRegex = /^\s*image:\s*(?<image>[^\s]+)/;

  for (const line of text.split("\n")) {
    const match = line.match(imageRegex);
    if (match?.groups?.image) {
      const img = match.groups.image.replace(/['"]/g, "");
      allImages.push(img);
    }
  }

  const unique = [...new Set(allImages)];

  if (filter) {
    colorLog(`Applying filter: *${filter}*`, "magenta");
    return unique.filter((img) => img.includes(filter));
  }

  return unique;
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
  if (existsSync(archiveFile) && !opts.overwrite) {
    colorLog(`File already exists, skipping: ${archiveFile}`, "yellow");
    return { success: true, archiveFile, repoPath, imageName: image };
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
  colorLog(`Downloading image: ${image}...`, "green");
  let proc;
  try {
    proc = Bun.spawn(["skopeo", ...skopeoArgs], {
      stdout: "pipe",
      stderr: "pipe",
    });
  } catch (e: any) {
    colorLog(`skopeo not found or not in PATH: ${e.message}`, "red");
    return { success: false, archiveFile, repoPath, imageName: image };
  }

  // Stream stdout and stderr for real-time progress
  const decoder = new TextDecoder();
  let stderrOutput = "";

  const streamOutput = async (stream: ReadableStream, toStderr: boolean) => {
    const reader = stream.getReader();
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        const text = decoder.decode(value, { stream: true });
        if (toStderr) {
          stderrOutput += text;
          process.stderr.write(text);
        } else {
          process.stdout.write(text);
        }
      }
    } catch {
      // Stream may already be closed
    }
  };

  await Promise.all([
    streamOutput(proc.stdout, false),
    streamOutput(proc.stderr, true),
  ]);

  const exitCode = await proc.exited;

  if (exitCode !== 0) {
    colorLog(`Failed to download image: ${image}`, "red");
    if (stderrOutput) colorLog(stderrOutput, "red");
    // Clean up partial file
    if (existsSync(archiveFile)) {
      unlinkSync(archiveFile);
    }
    return { success: false, archiveFile, repoPath, imageName: image };
  }

  colorLog(`Downloaded: ${image} -> ${archiveFile}`, "green");
  return { success: true, archiveFile, repoPath, imageName: image };
}

export function parseExistingUploadScript(scriptPath: string): string[] {
  if (!existsSync(scriptPath)) return [];

  const content = readFileSync(scriptPath, "utf-8");
  const entries: string[] = [];
  const entryRegex = /^\s*"([^"]+?\|docker:\/\/[^"]+)"$/;

  for (const line of content.split("\n")) {
    const match = line.match(entryRegex);
    if (match?.[1]) {
      entries.push(match[1]);
    }
  }

  return entries;
}

export function generateUploadScript(
  entries: DownloadResult[],
  targetDir: string,
  skopeoPath: string | null,
  registry: string
): void {
  const successful = entries.filter((e) => e.success);
  if (successful.length === 0) return;

  const scriptPath = `${targetDir}/upload_all.ps1`;

  // Read existing entries to preserve them
  const existingEntries = parseExistingUploadScript(scriptPath);

  // Build new entries from current download
  const newEntries: string[] = [];
  for (const entry of successful) {
    const fileName = entry.archiveFile.split("/").pop();
    newEntries.push(`${fileName}|docker://${registry}/${entry.repoPath}`);
  }

  // Merge: existing + new (no duplicates)
  const allEntries = [...existingEntries];
  for (const newEntry of newEntries) {
    if (!allEntries.includes(newEntry)) {
      allEntries.push(newEntry);
    }
  }

  const isWin = process.platform === "win32";
  const skopeoCmd = skopeoPath ? `"${skopeoPath}"` : "skopeo";

  const lines: string[] = [
    "# Auto-generated upload script by skopeo-cli",
    "# Place this script in the same directory as the image files",
    "",
    "Set-Location $PSScriptRoot",
    "",
    "$images = @(",
  ];

  for (const entry of allEntries) {
    lines.push(`    "${entry}"`);
  }

  lines.push(");",
    "",
    "for ($i = 0; $i -lt $images.Count; $i++) {",
    "    $parts = $images[$i] -split '\\|'",
    `    Write-Host "[$($i+1)/$($images.Count)] Uploading: $($parts[0]) ..." -ForegroundColor Cyan`,
    `    & ${skopeoCmd} copy --all "oci-archive:$($parts[0])" $parts[1]`,
    "    if ($LASTEXITCODE -ne 0) {",
    "        Write-Host \"Upload failed: $($parts[0])\" -ForegroundColor Red",
    "    }",
    "}",
    "Write-Host \"All uploads completed!\" -ForegroundColor Green",
  );

  const scriptContent = lines.join("\n");
  Bun.write(scriptPath, scriptContent);
  colorLog(`Generated upload script: ${scriptPath}`, "green");
}

// ── Commands ───────────────────────────────────────────────────────────────

async function composeCommand(
  file: string,
  opts: ComposeOptions
): Promise<void> {
  colorLog(`Parsing ${file} ...`, "cyan");
  const images = parseComposeFile(file, opts.filter);

  if (images.length === 0) {
    colorLog("No images found.", "yellow");
    return;
  }

  const savePath = opts.savePath || getDefaultDownloadPath();
  const outputDir = `${savePath}/docker-image-will-upload`;
  ensureDir(outputDir);

  // Check for skopeo availability
  const skopeoPath = await findSkopeo();
  if (!skopeoPath) {
    colorLog("Error: skopeo not found. Please install skopeo first.", "red");
    colorLog("  Windows: https://github.com/containers/skopeo/blob/main/install.md", "white");
    colorLog("  Linux:   sudo apt install skopeo / sudo yum install skopeo", "white");
    process.exit(1);
  }
  colorLog(`Found skopeo: ${skopeoPath}`, "cyan");

  // Copy skopeo to output directory
  const isWin = process.platform === "win32";
  if (isWin) {
    colorLog("Copying skopeo to output directory...", "cyan");
    await copySkopeoToDir(outputDir);
  }

  const effectiveOpts: DownloadOptions = { ...opts, savePath: outputDir };

  colorLog(`Found ${images.length} matching images, starting download...`, "cyan");
  console.log("==================================================");

  const results: DownloadResult[] = [];
  for (let i = 0; i < images.length; i++) {
    const img = images[i]!;
    colorLog(`[${i + 1}/${images.length}] Processing: ${img}`, "cyan");

    const result = await downloadImage(img, effectiveOpts);
    results.push(result);

    if (!result.success) {
      colorLog(`Warning: ${img} download failed.`, "yellow");
    }
    console.log("--------------------------------------------------");
  }

  const succeeded = results.filter((r) => r.success).length;
  const failed = results.length - succeeded;
  if (failed > 0) {
    colorLog(`Done! Success: ${succeeded}, Failed: ${failed}`, "yellow");
  } else {
    colorLog("All downloads completed!", "green");
  }

  if (!opts.noUploadScript) {
    generateUploadScript(results, outputDir, isWin ? skopeoPath : null, opts.registry);
  }

  colorLog(`Output directory: ${outputDir}`, "cyan");
}

async function downloadCommand(
  image: string,
  opts: DownloadOptions
): Promise<void> {
  const savePath = opts.savePath || getDefaultDownloadPath();
  const outputDir = `${savePath}/docker-image-will-upload`;
  ensureDir(outputDir);

  // Check for skopeo availability
  const skopeoPath = await findSkopeo();
  if (!skopeoPath) {
    colorLog("Error: skopeo not found. Please install skopeo first.", "red");
    colorLog("  Windows: https://github.com/containers/skopeo/blob/main/install.md", "white");
    colorLog("  Linux:   sudo apt install skopeo / sudo yum install skopeo", "white");
    process.exit(1);
  }
  colorLog(`Found skopeo: ${skopeoPath}`, "cyan");

  // Copy skopeo to output directory
  const isWin = process.platform === "win32";
  if (isWin) {
    colorLog("Copying skopeo to output directory...", "cyan");
    await copySkopeoToDir(outputDir);
  }

  const effectiveOpts: DownloadOptions = { ...opts, savePath: outputDir };

  const result = await downloadImage(image, effectiveOpts);

  if (result.success) {
    colorLog(`Recorded: ${image} -> ${opts.registry}/${result.repoPath}`, "green");
    if (!opts.noUploadScript) {
      generateUploadScript([result], outputDir, isWin ? skopeoPath : null, opts.registry);
    }
  }

  colorLog(`Output directory: ${outputDir}`, "cyan");
}

// ── Main entry ─────────────────────────────────────────────────────────────

if (import.meta.main) {
const args = process.argv.slice(2);
const parsed = parseArgs(args);

switch (parsed.command) {
  case "download": {
    const image = parsed.positional[0];
    if (!image) {
      colorLog("Usage: skopeo-cli download <image>", "yellow");
      process.exit(1);
    }
    const opts: DownloadOptions = {
      savePath: (parsed.options.save as string) || "",
      platform: parsed.options.platform as string | undefined,
      overwrite: !!parsed.options.overwrite,
      noUploadScript: !!parsed.options["no-upload-script"],
      registry: (parsed.options.registry as string) || "docker.senjone.com",
    };
    await downloadCommand(image, opts);
    break;
  }
  case "compose": {
    const file = parsed.positional[0];
    if (!file) {
      colorLog("Usage: skopeo-cli compose <file>", "yellow");
      process.exit(1);
    }
    const opts: ComposeOptions = {
      savePath: (parsed.options.save as string) || "",
      filter: parsed.options.filter as string | undefined,
      overwrite: !!parsed.options.overwrite,
      noUploadScript: !!parsed.options["no-upload-script"],
      registry: (parsed.options.registry as string) || "docker.senjone.com",
    };
    await composeCommand(file, opts);
    break;
  }
  default:
    colorLog("Usage: skopeo-cli <download|compose> [options]", "yellow");
    colorLog("  download <image>    Download a single Docker image", "white");
    colorLog("  compose <file>      Batch download from compose file", "white");
    colorLog("  --registry <host>   Target registry (default: docker.senjone.com)", "white");
    process.exit(1);
}
}
