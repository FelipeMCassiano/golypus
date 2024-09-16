GPid =$(cat golypus.pid)

echo $GPid

sudo kill -SIGTERM $GPid

sudo rm golypus.log
sudo rm golypus.pid

clear

echo "Starting golypus..."

sudo go run main.go

echo "Recieving logs..."
echo ""

sudo tail -f golypus.log
