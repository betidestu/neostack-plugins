@echo off
REM Configures Claude Code and Codex MCP configs for Windows.
REM Codex needs an absolute proxy path so the process cwd stays on the active
REM Unreal project instead of the plugin's bin directory.

setlocal
set "DIR=%~dp0"

if not exist "%DIR%codex-windows.mcp.json" (
  echo Missing codex-windows.mcp.json ? was the plugin built with all targets? 1>&2
  exit /b 1
)
if not exist "%DIR%claude-windows.mcp.json" (
  echo Missing claude-windows.mcp.json ? was the plugin built with all targets? 1>&2
  exit /b 1
)

copy /Y "%DIR%claude-windows.mcp.json" "%DIR%.mcp.json" >nul
set "PROXY=%DIR%bin\win64\neostack-mcp-proxy.exe"
(
  echo {
  echo   "mcpServers": {
  echo     "neostack": {
  echo       "command": "%PROXY:\=\\%"
  echo     }
  echo   }
  echo }
) > "%DIR%codex.mcp.json"

where codex >nul 2>nul
if %ERRORLEVEL% EQU 0 (
  codex mcp add neostack -- "%PROXY%" >nul
  echo Registered neostack with Codex using an absolute proxy path.
) else (
  echo Codex CLI not found on PATH; generated codex.mcp.json only.
)

echo Configured .mcp.json and codex.mcp.json for Windows. Run /reload-plugins or restart the app.
