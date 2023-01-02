## Ausführen

set GOTMPDIR=C:\MYCOMP
go run .\cmd\web
npm run dev

npm run build


nodemon --watch './**/*.go' --signal SIGTERM --exec 'go' run cmd/MyProgram/main.go