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

SET PATH=%PATH%;C:\Windows\System32
set GOTMPDIR=C:\MYCOMP
nodemon --delay 1s -e go,html --ignore web/app/ --signal SIGKILL --exec go run ./cmd/ncDocConverth || exit 1