const fs = require("fs");
const os = require("os");
const path = require("path");
const { execSync } = require("child_process");

const VERSION = require("../package.json").version;
const REPO = "loadingmans/pdf_cli";
const NAME = "pdf-cli";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];

if (!platform || !arch) {
  console.error(`Unsupported platform: ${process.platform}-${process.arch}`);
  process.exit(1);
}

const isWindows = process.platform === "win32";
const ext = isWindows ? ".zip" : ".tar.gz";
const archiveName = `${NAME}-${VERSION}-${platform}-${arch}${ext}`;
const releaseURL = `https://gitee.com/${REPO}/releases/download/v${VERSION}/${archiveName}`;

const binDir = path.join(__dirname, "..", "bin");
const dest = path.join(binDir, NAME + (isWindows ? ".exe" : ""));

fs.mkdirSync(binDir, { recursive: true });

function download(url, destPath) {
  const sslFlag = isWindows ? "--ssl-revoke-best-effort " : "";
  execSync(
    `curl ${sslFlag}--fail --location --silent --show-error --connect-timeout 10 --max-time 120 --output "${destPath}" "${url}"`,
    { stdio: ["ignore", "ignore", "pipe"] }
  );
}

function extract(archivePath, tmpDir) {
  if (isWindows) {
    execSync(
      `powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${tmpDir}' -Force"`,
      { stdio: "ignore" }
    );
    return;
  }

  execSync(`tar -xzf "${archivePath}" -C "${tmpDir}"`, { stdio: "ignore" });
}

function install() {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "pdf-cli-"));
  const archivePath = path.join(tmpDir, archiveName);
  const binaryName = NAME + (isWindows ? ".exe" : "");
  const extractedBinary = path.join(tmpDir, binaryName);

  try {
    download(releaseURL, archivePath);
    extract(archivePath, tmpDir);
    fs.copyFileSync(extractedBinary, dest);
    if (!isWindows) {
      fs.chmodSync(dest, 0o755);
    }
    console.log(`${NAME} v${VERSION} installed successfully`);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

try {
  install();
} catch (error) {
  console.error(`Failed to install ${NAME}:`, error.message);
  console.error(
    "\nRelease asset not found or download failed. Confirm that the matching Gitee Release exists before publishing this npm version."
  );
  process.exit(1);
}
