# msmareqapp
docker build --no-cache -t msmareqapp .
docker run -dp 9088:8080 --name msmareqapp_dev msmareqapp