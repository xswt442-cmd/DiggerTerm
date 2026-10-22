# DiggerTerm Architecture

English · [简体中文](ARCHITECTURE.zh-CN.md)

## Components and Ownership

DiggerTerm is a Go/Wails 3 desktop terminal using Vue 3/TypeScript and xterm.js for its interface. The same Go application runs as the GUI or as a separate `--daemon` process.

- Root Go application: startup, embedded resources, and GUI/Daemon mode selection.
- `frontend/`: terminal view, session sidebar, splits, search, settings, and calls into Wails services.
- `internal/desktop/`: services exposed to the GUI, user preferences, and GUI lifecycle; uses IPC to interact with the Daemon.
- `internal/daemon/`: managed sessions, OS processes, and IPC; owns PTY/ConPTY, WSL, and SSH client operations.
- `internal/session/`: managed session IO, resize, output replay, and lifecycle.
- `internal/observe/`: read-only external Windows/WSL process metadata, source-aware merging, and process identity handling.

## Sessions and IPC Boundaries

The GUI is an IPC client of the Daemon. Closing, hiding, or reopening the GUI can leave Daemon-managed sessions alive and reconnectable; terminating the Daemon or the operating system does not preserve active processes. IPC is scoped to the current OS user, and the GUI cannot bypass the Daemon to control terminals directly.

Quick CD is a GUI-side, explicitly triggered action on the active managed PTY: it validates and quotes a directory path for the selected Shell (PowerShell, CMD, or POSIX), then queues the text through the same controlled input path. No newline is sent, so a foreground application cannot be assumed to be an idle shell. The operator presses Enter at a shell prompt; it is not an OS-level remote working-directory mutation. A local absolute path may optionally update the existing new-session default-directory preference. Observed external processes have no Quick CD route.

A running managed session has only one GUI controller at a time. The Daemon enforces the lease for input, resize, and termination: another GUI cannot terminate a session held by a controller, while an unowned managed session can be explicitly terminated. Closing a pane only detaches its view; accepted terminal input is queued per session and sent before the detach request. Explicit bulk termination rechecks the target set and follows confirmation rules.

GUI IPC calls have bounded deadlines: 15 seconds for ordinary methods, 30 seconds for session creation, and 45 seconds for external process scans. A timeout means the client has not received the result; the Daemon might still complete the action. The client must query session state before retrying creation or termination to avoid duplicate actions.

## External Observation and Platform Boundaries

`Managed` sessions are created and owned by the Daemon. `Observed` processes are external terminal or Agent processes whose metadata is read-only. External observation does not grant PTY screen access, input injection, process termination, or takeover.

Windows and WSL distribution PIDs belong to different namespaces. External process parent-child relations can be incomplete, so the UI identifies inferred relationships accordingly. Windows ConPTY, WSL, OpenSSH, and Linux PTY behavior are implemented behind platform-specific code.

## Data and Settings

Terminal layout, themes, shortcuts, and preferences use configuration directories under the current user's account. Users can provide JSON/CSS themes; the Help page or source of the configuration service identifies the actual platform-specific paths. Documentation does not embed one developer machine's private directories.

## Source and Release

Source, tests, and build commands are maintained with their respective modules and workflows. PR CI uses `scripts/check-docs.mjs` to verify documentation links, language pairs, bilingual CHANGELOG categories, and current version alignment; PR template synchronization is also checked. GitHub Release notes are generated from the tagged CHANGELOG version. See the [release runbook](RELEASE.md). Public documentation describes current behavior and real limitations, not individual test runs or private environments.
