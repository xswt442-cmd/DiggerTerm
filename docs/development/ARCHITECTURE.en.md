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

A managed session has only one GUI controller at a time. Closing a terminal pane detaches its view. Explicit bulk session termination rechecks the target set and follows confirmation rules.

## External Observation and Platform Boundaries

`Managed` sessions are created and owned by the Daemon. `Observed` processes are external terminal or Agent processes whose metadata is read-only. External observation does not grant PTY screen access, input injection, process termination, or takeover.

Windows and WSL distribution PIDs belong to different namespaces. External process parent-child relations can be incomplete, so the UI identifies inferred relationships accordingly. Windows ConPTY, WSL, OpenSSH, and Linux PTY behavior are implemented behind platform-specific code.

## Data and Settings

Terminal layout, themes, shortcuts, and preferences use configuration directories under the current user's account. Users can provide JSON/CSS themes; the Help page or source of the configuration service identifies the actual platform-specific paths. Documentation does not embed one developer machine's private directories.

## Source and Release

Source, tests, and build commands are maintained with their respective modules and workflows. GitHub Release notes are generated from the tagged CHANGELOG version. See the [release runbook](RELEASE.md). Public documentation describes current behavior and real limitations, not individual test runs or private environments.
