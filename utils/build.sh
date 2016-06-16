GOOS=linux GOARCH=amd64 go build -o stout-linux src/* 
GOOS=darwin GOARCH=amd64 go build -o stout-osx src/* 
GOOS=windows GOARCH=amd64 go build -o stout-windows.exe src/* 
