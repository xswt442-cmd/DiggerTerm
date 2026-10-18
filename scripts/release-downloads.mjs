import { readdirSync, readFileSync, statSync } from 'node:fs';
import { basename, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const VERSION_TAG = /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/;
const REPOSITORY = /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;

export function releaseDownloadSection(tag, repository, assetNames) {
  const match = VERSION_TAG.exec(tag);
  if (!match) throw new Error('tag must use vX.Y.Z');
  if (!REPOSITORY.test(repository)) throw new Error('repository must be owner/name');
  if (!Array.isArray(assetNames) || assetNames.some((name) => typeof name !== 'string'
    || name !== basename(name) || name.includes('\\') || !/^[A-Za-z0-9_.-]+$/.test(name))) {
    throw new Error('release assets must be safe file names');
  }

  const portable = tag === 'v0.0.1';
  const rows = portable
    ? [['Windows x64', 'ZIP (portable)', `DiggerTerm-${tag}-windows-amd64.zip`]]
    : [
      ['Windows x64', 'EXE', `DiggerTerm-${tag}-windows-amd64.exe`],
      ['Linux x64', 'AppImage', `DiggerTerm-${tag}-linux-x86_64.AppImage`],
      ['Ubuntu / Debian x64', 'DEB', `DiggerTerm-${tag}-linux-amd64.deb`],
    ];
  const checksum = portable ? `DiggerTerm-${tag}-windows-amd64.zip.sha256` : 'SHA256SUMS';
  const required = [...rows.map((row) => row[2]), checksum];
  if (new Set(assetNames).size !== required.length || assetNames.length !== required.length
    || required.some((name) => !assetNames.includes(name))) {
    throw new Error('release assets do not match the expected packages and checksum for ' + tag);
  }
  const url = (name) => `https://github.com/${repository}/releases/download/${tag}/${name}`;
  const table = [
    '## 下载 / Downloads',
    '',
    '| 平台 / Platform | 格式 / Package | 下载 / Download |',
    '| --- | --- | --- |',
    ...rows.map(([platform, format, filename]) => `| ${platform} | ${format} | [下载 / Download](${url(filename)}) |`),
    `| 校验 / Checksums | SHA-256 | [${checksum}](${url(checksum)}) |`,
    '',
    '**数字签名 / Signatures:** Windows 发行包未进行代码签名；本版没有随附发行者数字签名。SHA-256 仅验证文件完整性，不代表发布者身份。 / Windows packages are unsigned; no publisher digital signature is provided. SHA-256 checks integrity, not publisher identity.',
    '',
  ];
  return table.join('\n');
}

function cliArgs(argv) {
  if (argv.length !== 6) {
    throw new Error('usage: node scripts/release-downloads.mjs --tag TAG --repo OWNER/REPO --dir DIR (or --assets-json FILE)');
  }
  const args = {};
  for (let i = 0; i < argv.length; i += 2) {
    if (!['--tag', '--repo', '--dir', '--assets-json'].includes(argv[i])
      || Object.hasOwn(args, argv[i]) || !argv[i + 1]) throw new Error('invalid argument: ' + argv[i]);
    args[argv[i]] = argv[i + 1];
  }
  if (!args['--tag'] || !args['--repo'] || Boolean(args['--dir']) === Boolean(args['--assets-json'])) {
    throw new Error('provide a tag, repository and exactly one asset source');
  }
  return args;
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const args = cliArgs(process.argv.slice(2));
    let assets;
    if (args['--assets-json']) {
      assets = JSON.parse(readFileSync(resolve(args['--assets-json']), 'utf8'));
    } else {
      const dir = resolve(args['--dir']);
      assets = readdirSync(dir);
      if (assets.some((name) => !statSync(resolve(dir, name)).isFile() || statSync(resolve(dir, name)).size === 0)) {
        throw new Error('release directory contains an empty or non-file asset');
      }
    }
    process.stdout.write(releaseDownloadSection(args['--tag'], args['--repo'], assets));
  } catch (error) {
    process.stderr.write('release-downloads: ' + error.message + '\n');
    process.exitCode = 1;
  }
}
