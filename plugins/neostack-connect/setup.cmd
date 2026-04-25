@echo off
REM Configures codex.mcp.json for Windows.
REM The shipped codex.mcp.json already targets Windows, so this just makes it explicit.

setlocal
set "DIR=%~dp0"

if not exist "%DIR%codex-windows.mcp.json" (
  echo Missing codex-windows.mcp.json — was the plugin built with all targets? 1>&2
  exit /b 1
)

copy /Y "%DIR%codex-windows.mcp.json" "%DIR%codex.mcp.json" >nul
echo Configured codex.mcp.json for Windows. Run `codex /reload-plugins` (or restart Codex).
