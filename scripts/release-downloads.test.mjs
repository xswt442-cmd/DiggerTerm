import test from 'node:test';
import assert from 'node:assert/strict';
import { releaseDownloadSection } from './release-downloads.mjs';

const repo = 'xswt442-cmd/DiggerTerm';

const modern = (tag) => [
  `DiggerTerm-${tag}-windows-amd64.exe`,
  `DiggerTerm-${tag}-linux-x86_64.AppImage`,
  `DiggerTerm-${tag}-linux-amd64.deb`,
  'SHA256SUMS',
];

test('modern releases contain real platform links, checksums, and signing status at the end', () => {
  const notes = releaseDownloadSection('v0.0.6', repo, modern('v0.0.6'));
  assert.match(notes, /^## 下载 \/ Downloads\n/);
  assert.match(notes, /releases\/download\/v0\.0\.6\/DiggerTerm-v0\.0\.6-windows-amd64\.exe/);
  assert.match(notes, /releases\/download\/v0\.0\.6\/DiggerTerm-v0\.0\.6-linux-x86_64\.AppImage/);
  assert.match(notes, /\| 校验 \/ Checksums \| SHA-256 \| \[SHA256SUMS\]/);
  assert.match(notes, /Windows packages are unsigned/);
  assert.match(notes, /SHA-256 checks integrity, not publisher identity/);
  assert.doesNotMatch(notes, /v0\.0\.5/);
});

test('first release describes only the published Windows ZIP and its checksum file', () => {
  const tag = 'v0.0.1';
  const zip = `DiggerTerm-${tag}-windows-amd64.zip`;
  const notes = releaseDownloadSection(tag, repo, [zip, zip + '.sha256']);
  assert.match(notes, /ZIP \(portable\)/);
  assert.match(notes, /\.zip\.sha256/);
  assert.doesNotMatch(notes, /AppImage|\.deb|SHA256SUMS/);
});

test('missing, extra, mismatched, or unsafe assets fail closed', () => {
  const assets = modern('v0.0.6');
  assert.throws(() => releaseDownloadSection('v0.0.6', repo, assets.slice(1)), /do not match/);
  assert.throws(() => releaseDownloadSection('v0.0.6', repo, [...assets, 'extra.exe']), /do not match/);
  assert.throws(() => releaseDownloadSection('v0.0.5', repo, assets), /do not match/);
  assert.throws(() => releaseDownloadSection('v0.0.6', repo, [...assets.slice(0, 3), '../unsafe']), /safe file names/);
  assert.throws(() => releaseDownloadSection('v0.0.6', 'bad repo', assets), /owner\/name/);
  assert.throws(() => releaseDownloadSection('v1.2', repo, assets), /vX.Y.Z/);
});
