sudo rm golypus.log
sudo rm golypus.pid

clear

echo "Starting program"

sudo go run main.go

echo "Recieving logs"

sudo tail -f golypus.log
