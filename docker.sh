GOOS=linux GOARCH=amd64 go build -o ./bin/rsshub-refresh main.go
docker build -t beegedelow/rsshub-refresh:latest .
docker push beegedelow/rsshub-refresh:latest