import test from 'node:test';
import assert from 'node:assert/strict';
import { addReleaseDownloads, checkManifestVersion, extractReleaseNotes, versionFromTag } from './release-notes.mjs';

const changelog = [
  '# Changelog',
  '',
  '## [Unreleased]',
  '',
  '### 简体中文',
  '',
  '### English',
  '',
  '## [1.2.4] - 2026-10-18',
  '',
  '### 简体中文',
  '',
  '- 修复列表筛选。',
  '',
  '### English',
  '',
  '- Fixed list filtering.',
  '',
  '## [1.2.3] - 2026-10-17',
  '',
  '### 简体中文',
  '',
  '- 新增分页。',
  '',
  '### English',
  '',
  '- Added pagination.',
  '',
].join('\n');

test('extracts exactly the target version in two languages', () => {
  const release = extractReleaseNotes(changelog, 'v1.2.4');
  assert.match(release, /^# v1\.2\.4\n/);
  assert.doesNotMatch(release, /更新说明|Release highlights/);
  assert.match(release, /修复列表筛选/);
  assert.match(release, /Fixed list filtering/);
  assert.doesNotMatch(release, /新增分页/);
  assert.doesNotMatch(release, /Unreleased/);
});

test('places localized downloads inside their matching language sections', () => {
  const notes = extractReleaseNotes(changelog, 'v1.2.4');
  const complete = addReleaseDownloads(notes, 'v1.2.4', 'owner/repo', [
    'DiggerTerm-v1.2.4-windows-amd64.exe',
    'DiggerTerm-v1.2.4-linux-x86_64.AppImage',
    'DiggerTerm-v1.2.4-linux-amd64.deb',
    'SHA256SUMS',
  ]);
  const chinese = complete.indexOf('### 简体中文');
  const english = complete.indexOf('### English');
  const chineseDownloads = complete.indexOf('### 下载');
  const englishDownloads = complete.indexOf('### Downloads');
  assert.ok(chinese < chineseDownloads && chineseDownloads < english);
  assert.ok(english < englishDownloads);
  assert.doesNotMatch(complete, /## 下载 \/ Downloads/);
  assert.doesNotMatch(complete, /平台 \/ Platform|下载 \/ Download/);
  assert.throws(() => addReleaseDownloads(complete, 'v1.2.4', 'owner/repo', []), /existing download section/);
});

test('supports product-scoped tags', () => {
  assert.equal(versionFromTag('desktop-v1.2.3'), '1.2.3');
  assert.match(extractReleaseNotes(changelog, 'cli-v1.2.3'), /Added pagination/);
});

test('rejects invalid tag and missing version', () => {
  assert.throws(() => versionFromTag('preview'), /tag must use/);
  assert.throws(() => extractReleaseNotes(changelog, 'v2.0.0'), /exactly one/);
});

test('rejects duplicate matching version headings', () => {
  assert.throws(() => extractReleaseNotes(changelog + '\n## [1.2.4]\n', 'v1.2.4'), /exactly one/);
});

test('rejects missing or reversed bilingual sections', () => {
  assert.throws(() => extractReleaseNotes(changelog.replace('### English\n\n- Fixed list filtering.', '### 英语\n\n- Fixed list filtering.'), 'v1.2.4'), /English/);
  assert.throws(() => extractReleaseNotes(changelog.replace('### 简体中文\n\n- 修复列表筛选。', '### 中文\n\n- 修复列表筛选。'), 'v1.2.4'), /简体中文/);
  const reversed = changelog.replace('### 简体中文\n\n- 修复列表筛选。\n\n### English\n\n- Fixed list filtering.', '### English\n\n- Fixed list filtering.\n\n### 简体中文\n\n- 修复列表筛选。');
  assert.throws(() => extractReleaseNotes(reversed, 'v1.2.4'), /precede/);
});

test('rejects empty language and template placeholders', () => {
  const empty = changelog.replace('- 修复列表筛选。', '');
  assert.throws(() => extractReleaseNotes(empty, 'v1.2.4'), /empty/);
  const draft = changelog.replace('- 修复列表筛选。', '<!-- insert changes -->');
  assert.throws(() => extractReleaseNotes(draft, 'v1.2.4'), /template comments/);
  assert.throws(() => extractReleaseNotes(changelog.replace('- 修复列表筛选。', '- TODO item'), 'v1.2.4'), /placeholder/);
});

test('enforces optional product manifest version', () => {
  checkManifestVersion('{"version":"1.2.4"}', 'v1.2.4');
  assert.throws(() => checkManifestVersion('{"version":"1.2.3"}', 'v1.2.4'), /does not match/);
});

test('does not include trailing changelog compare reference definitions', () => {
  const withLinks = changelog + '[Unreleased]: https://example.com/compare/v1.2.4...main\n[1.2.3]: https://example.com/compare/v1.2.2...v1.2.3\n';
  const release = extractReleaseNotes(withLinks, 'v1.2.3');
  assert.match(release, /Added pagination/);
  assert.doesNotMatch(release, /https:\/\/example.com/);
});
