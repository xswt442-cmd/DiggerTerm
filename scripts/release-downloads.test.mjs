import test from 'node:test';
import assert from 'node:assert/strict';
import { releaseDownloadSections } from './release-downloads.mjs';

const repo = 'xswt442-cmd/DiggerTerm';

const modern = (tag) => [
  `DiggerTerm-${tag}-windows-amd64.exe`,
  `DiggerTerm-${tag}-linux-x86_64.AppImage`,
  `DiggerTerm-${tag}-linux-amd64.deb`,
  'SHA256SUMS',
];

test('modern releases provide localized download sections and signing status', () => {
  const sections = releaseDownloadSections('v0.0.6', repo, modern('v0.0.6'));
  assert.match(sections.chinese, /^### 下载\n/);
  assert.match(sections.chinese, /releases\/download\/v0\.0\.6\/DiggerTerm-v0\.0\.6-windows-amd64\.exe/);
  assert.match(sections.chinese, /releases\/download\/v0\.0\.6\/DiggerTerm-v0\.0\.6-linux-x86_64\.AppImage/);
  assert.match(sections.chinese, /\| 校验文件 \| SHA-256 \| \[SHA256SUMS\]/);
  assert.match(sections.chinese, /签名状态/);
  assert.match(sections.english, /^### Downloads\n/);
  assert.match(sections.english, /\| Checksums \| SHA-256 \| \[SHA256SUMS\]/);
  assert.match(sections.english, /Windows packages are unsigned/);
  assert.match(sections.english, /SHA-256 checks integrity, not publisher identity/);
  assert.doesNotMatch(sections.chinese + sections.english, /v0\.0\.5/);
  assert.doesNotMatch(sections.chinese, /Platform|Download|Signing status/);
  assert.doesNotMatch(sections.english, /平台|下载|签名状态/);
});

test('first release describes only the published Windows ZIP and its checksum file', () => {
  const tag = 'v0.0.1';
  const zip = `DiggerTerm-${tag}-windows-amd64.zip`;
  const sections = releaseDownloadSections(tag, repo, [zip, zip + '.sha256']);
  assert.match(sections.chinese, /ZIP \(portable\)/);
  assert.match(sections.english, /ZIP \(portable\)/);
  assert.match(sections.chinese, /\.zip\.sha256/);
  assert.doesNotMatch(sections.chinese + sections.english, /AppImage|\.deb|SHA256SUMS/);
});

test('missing, extra, mismatched, or unsafe assets fail closed', () => {
  const assets = modern('v0.0.6');
  assert.throws(() => releaseDownloadSections('v0.0.6', repo, assets.slice(1)), /do not match/);
  assert.throws(() => releaseDownloadSections('v0.0.6', repo, [...assets, 'extra.exe']), /do not match/);
  assert.throws(() => releaseDownloadSections('v0.0.5', repo, assets), /do not match/);
  assert.throws(() => releaseDownloadSections('v0.0.6', repo, [...assets.slice(0, 3), '../unsafe']), /safe file names/);
  assert.throws(() => releaseDownloadSections('v0.0.6', 'bad repo', assets), /owner\/name/);
  assert.throws(() => releaseDownloadSections('v1.2', repo, assets), /vX.Y.Z/);
});
