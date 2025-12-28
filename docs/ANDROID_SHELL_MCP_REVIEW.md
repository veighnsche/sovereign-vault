# Android-Shell MCP Server Review

> **TEAM_007**: Review based on hands-on debugging session for AVF VM networking.
> **Audience**: MCP Developer Team
> **Purpose**: Improve the android-shell MCP server experience for AI agents

---

## Executive Summary

The android-shell MCP server is **functional and usable**, but has significant room for improvement in **output cleanliness**, **command batching**, and **reducing round-trip overhead**. During a single debugging session, I made **25+ sequential `run_command` calls** that could have been reduced to **5-8 batch calls** with proper support.

---

## Tool Usage Statistics (This Session)

| Tool | Times Used | Notes |
|------|------------|-------|
| `mcp0_run_command` | **25+** | By far the most used |
| `mcp0_quick_command` | 5 | Initial exploration |
| `mcp0_start_shell` | 1 | Shell setup |
| `mcp0_list_devices` | 1 | Device discovery |

**Conclusion**: `run_command` dominates usage. Optimizing it would have the highest impact.

---

## Critical Issues

### 1. Output Noise (HIGH PRIORITY)

The command output is polluted with internal markers:

```
osetup /dev/block/loop48 /data/sovereign/vm/s                         <ql/rootfs.img && mount /dev/block
/data/sovereign/vm/sql/rootfs.img && mount /dev/block                         </loop48 /data/local/tmp/r
echo "__EXIT_1adb3987__$?__EXIT_1adb3987__"; e                         <cho "___MCP_MARKER___287a
```

**What I actually wanted:**
```
Mounted
```

**Suggestion**: Strip all internal markers before returning output to the caller. Provide a `raw_output` field if the markers are ever needed for debugging.

---

### 2. No Command Batching (HIGH PRIORITY)

I made 25+ sequential tool calls like:
```
run_command("ls /path")
run_command("cat /path/file1")
run_command("cat /path/file2")
run_command("grep something /path/file3")
```

**What I wanted:**
```javascript
run_commands([
  { command: "ls /path", id: "list" },
  { command: "cat /path/file1", id: "file1" },
  { command: "cat /path/file2", id: "file2" },
  { command: "grep something /path/file3", id: "grep_result" }
])
// Returns: { list: {...}, file1: {...}, file2: {...}, grep_result: {...} }
```

**Benefits**:
- Reduces round-trip latency dramatically
- Reduces token usage (no repeated tool call overhead)
- AI can plan ahead and batch independent operations

---

### 3. Unknown Exit Codes

Sometimes the tool returns:
```
STATUS: COMPLETED
EXIT_CODE: unknown
```

This is confusing. Exit codes should always be captured. If the shell session died, say so explicitly.

---

### 4. Shell Echo/Prompt Leakage

Command echoing pollutes output:
```
oriole:/data/sovereign/vm/sql # cd /data/sovereign/vm/sql && sh start.sh; echo
d /data/sovereign/vm/sql && sh start.sh; echo                                 < "__EXIT_8fc5a1ce__$?__EXIT_8fc5a
```

The prompt and command echo should be stripped from output.

---

## Requested Features

### Priority 1: Batch Commands

```javascript
// New tool: run_commands (plural)
{
  "name": "run_commands",
  "parameters": {
    "shell_id": "string",
    "commands": [
      {
        "id": "string",           // Optional identifier for this command
        "command": "string",       // The command to run
        "timeout_seconds": 30,     // Per-command timeout
        "continue_on_error": true, // Whether to run next command if this fails
        "capture_output": true     // Whether to include output in response
      }
    ],
    "stop_on_first_error": false   // Global flag
  }
}
```

**Example use case** (my actual workflow):
```javascript
run_commands({
  shell_id: "root_shell",
  commands: [
    { id: "stop_vm", command: "pkill -f 'crosvm.*sql'" },
    { id: "fsck", command: "e2fsck -fy /data/.../rootfs.img", timeout_seconds: 60 },
    { id: "mount", command: "losetup /dev/block/loop48 ... && mount ..." },
    { id: "create_devs", command: "mknod /dev/console c 5 1 && mknod /dev/vsock c 10 121" },
    { id: "unmount", command: "sync && umount -l ..." },
    { id: "start_vm", command: "sh start.sh" }
  ]
})
```

This would have replaced **10 separate tool calls** with **1 call**.

---

### Priority 2: Clean Output Mode

Add a parameter to strip all markers:

```javascript
run_command({
  shell_id: "...",
  command: "cat /file",
  clean_output: true  // Strip markers, prompts, command echo
})
```

Or make clean output the default and add `include_debug_markers: true` for debugging.

---

### Priority 3: File Transfer Tools

Currently I have to do:
```javascript
run_command("cat /path/to/file | base64")
// Then decode the base64 myself
```

**Requested tools**:
```javascript
// Pull file from device
pull_file({
  device_serial: "...",
  remote_path: "/data/file",
  encoding: "utf8" | "base64"  // For binary files
})

// Push file to device
push_file({
  device_serial: "...",
  remote_path: "/data/file",
  content: "...",
  encoding: "utf8" | "base64"
})
```

---

### Priority 4: Conditional Command Chains

```javascript
run_command_chain({
  shell_id: "...",
  chain: [
    { command: "test -f /file", on_success: "next", on_failure: "skip_to:create" },
    { command: "cat /file" },
    { id: "create", command: "touch /file" }
  ]
})
```

This would let me express: "if file exists, read it; otherwise create it" in one call.

---

### Priority 5: Output Streaming / Pagination

For commands with large output:

```javascript
run_command({
  command: "cat /large/file",
  max_output_lines: 100,
  output_mode: "head" | "tail" | "paginated"
})
// Returns: { output: "...", truncated: true, total_lines: 5000, next_offset: 100 }
```

---

## Minor Improvements

### 6. Better Status Messages

Instead of:
```
STATUS: COMPLETED
EXIT_CODE: unknown
```

Provide:
```
STATUS: COMPLETED
EXIT_CODE: 0
SHELL_STATE: responsive | unresponsive | closed
OUTPUT_TRUNCATED: false
```

### 7. Command History

```javascript
get_shell_history({
  shell_id: "...",
  last_n: 10
})
// Returns last 10 commands and their results
```

Useful when I need to recall what I did earlier in the session.

### 8. Environment Variables

```javascript
set_env({
  shell_id: "...",
  variables: {
    "PATH": "/custom/path:$PATH",
    "MY_VAR": "value"
  }
})
```

### 9. Working Directory Management

Currently I can't `cd` effectively because each command might be independent. Add:

```javascript
run_command({
  shell_id: "...",
  working_directory: "/data/sovereign/vm/sql",  // Changes to this dir first
  command: "ls -la"
})
```

---

## What Works Well

To be fair, several things work great:

1. **`list_devices`** - Clean, informative output with device models
2. **`start_shell` / `stop_shell`** - Persistent sessions work well
3. **`quick_command`** - Great for one-off commands
4. **Root vs non-root shells** - Useful distinction
5. **Background jobs** - `run_background` + `check_job` is a good pattern
6. **Control characters** - `send_control_char` for Ctrl+C is essential
7. **Shell diagnosis** - `diagnose_shell` is helpful for debugging

---

## Implementation Priority

| Priority | Feature | Impact | Effort |
|----------|---------|--------|--------|
| 1 | **Batch commands** | Very High | Medium |
| 2 | **Clean output** | High | Low |
| 3 | **File transfer** | High | Medium |
| 4 | **Proper exit codes** | Medium | Low |
| 5 | **Conditional chains** | Medium | High |
| 6 | **Output pagination** | Low | Medium |

---

## Token/Efficiency Analysis

**Current session overhead:**
- 25 `run_command` calls
- Each call: ~200 tokens for tool invocation + ~300 tokens for response
- Total: ~12,500 tokens for shell interactions

**With batch commands:**
- 5-8 batch calls
- Each call: ~400 tokens for invocation + ~800 tokens for multi-result response
- Total: ~4,800 tokens

**Savings: ~60% token reduction**, plus faster execution due to reduced round-trips.

---

## Conclusion

The android-shell MCP server is a solid foundation. The **single biggest improvement** would be **batch command support** with clean output. This alone would transform the developer experience from "workable but tedious" to "efficient and pleasant."

Secondary priorities are **file transfer** and **proper exit code handling**.

---

*Document created: 2025-12-28 by TEAM_007*
*Based on: AVF VM networking debugging session*

---

## TEAM_011 Addendum: Why I Abandoned MCP for Manual Commands

> **Date**: 2025-12-28
> **Context**: Continuing VM networking work from TEAM_007

### The Drift Pattern

I started with MCP tools but gradually shifted to raw `adb` commands. Here's what happened:

#### Phase 1: Started with MCP (Good)
```
mcp0_list_devices()           # Found Pixel 6
mcp0_start_shell()            # Got root shell
mcp0_run_commands()           # Checked VM state
```

#### Phase 2: Hit Friction, Started Mixing
When I needed to push files, MCP's `file_transfer` tool felt limited:
- Only works for small files
- No progress indication for large transfers
- Easier to just: `adb push rootfs.img /data/...`

#### Phase 3: Full Manual Mode (Bad)
```bash
# I ended up doing everything manually:
docker build --platform linux/arm64 -t sovereign-sql ...
docker export ... > /tmp/rootfs.tar
sudo mount ... && sudo tar -xf ... && sudo mknod ...
adb push rootfs.img /data/sovereign/vm/sql/
adb shell su -c 'sh /data/sovereign/vm/sql/start.sh'
```

### Root Causes of Drift

| Cause | Description |
|-------|-------------|
| **1. File size limits** | MCP's `file_transfer` max is 1MB. Rootfs is 512MB. |
| **2. Docker operations** | No MCP tool for Docker - had to use run_command anyway |
| **3. Sudo operations** | Host-side sudo for mount/mknod not supported by MCP |
| **4. Cognitive load** | Easier to remember `adb push` than MCP tool parameters |
| **5. Iteration speed** | Raw commands felt faster when debugging |

### The Bigger Problem: Ignored Sovereign CLI Entirely

**This is the critical failure.** The sovereign CLI already had everything:

```bash
sovereign build --sql     # Docker build + rootfs export + device nodes
sovereign deploy --sql    # Push all files to device
sovereign start --sql     # Start VM with gvproxy
sovereign test --sql      # Verify Tailscale + PostgreSQL
sovereign prepare --sql   # Fix rootfs for AVF (idempotent)
```

I reinvented all of this manually:
- Manual `docker build` → should have used `sovereign build --sql`
- Manual `mknod` for device nodes → `rootfs.PrepareForAVF()` handles this
- Manual `adb push` → `sovereign deploy --sql`
- Manual shell scripts → `sovereign start --sql`

### Why Did I Ignore the CLI?

1. **Didn't check first** - Started working immediately without exploring the codebase
2. **NIH syndrome** - Felt I needed to "understand" by doing manually
3. **MCP tunnel vision** - Focused on MCP tools, forgot there's a Go CLI
4. **No memory prompt** - System didn't remind me about the CLI's existence

### Lessons Learned

1. **Always explore the existing tooling first** - `code_search` for CLI commands
2. **The MCP server is for device interaction, not builds** - Use the project's native tools
3. **Sovereign CLI should be the primary interface** - MCP for debugging only
4. **Document the correct workflow** - So future teams don't repeat this

### Correct Workflow (For Future Teams)

```bash
# 1. Build VM (uses Docker, creates rootfs with device nodes)
cd /home/vince/Projects/android/kernel/sovereign
go run ./cmd/sovereign build --sql

# 2. Deploy to device (pushes all files)
go run ./cmd/sovereign deploy --sql

# 3. Start VM (starts gvproxy + crosvm)
go run ./cmd/sovereign start --sql

# 4. Test connectivity
go run ./cmd/sovereign test --sql

# Only use MCP for:
# - Debugging inside the running VM
# - Checking logs when things fail
# - Interactive troubleshooting
```

### MCP Server Recommendations (Updated)

Given that the sovereign CLI exists, the MCP server's role changes:

| Use Case | Tool |
|----------|------|
| Build/Deploy/Start | `sovereign` CLI |
| Check device state | MCP `run_commands` |
| Read logs | MCP `run_command` |
| Debug networking | MCP shell session |
| File transfers | CLI (it uses `adb push` internally) |

**New Recommendation**: MCP should detect when a project has a CLI and suggest using it:

```
Warning: Found sovereign CLI at /home/.../sovereign
Consider using: go run ./cmd/sovereign <command>
Instead of raw adb commands.
```

---

*Addendum by TEAM_011, 2025-12-28*
