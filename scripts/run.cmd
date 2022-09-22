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

set GOTMPDIR=C:\MYCOMP
nodemon --delay 1s -e go,html --ignore web/app/ --exec go run ./cmd/ncDocConverth --signal SIGTERM