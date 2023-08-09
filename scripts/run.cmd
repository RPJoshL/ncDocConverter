@ECHO OFF

:: Bypass the "Terminate Batch Job" prompt
if "%~1"=="-FIXED_CTRL_C" (
   :: Remove the -FIXED_CTRL_C parameter
   SHIFT
) ELSE (
   :: Run the batch with <NUL and -FIXED_CTRL_C
   CALL <NUL %0 -FIXED_CTRL_C %*
   GOTO :EOF
)

SET PATH=%PATH%;C:\Windows\System3

set /p version=< VERSION

.\web\app\node_modules\.bin\nodemon --delay 1s -e go,html,yaml --signal SIGKILL --ignore web/app/ --quiet ^
--exec "echo [Restarting] && go run -ldflags ""-X main.version=%VERSION%"" ./cmd/ncDocConverth" -- %args% || "exit 1"