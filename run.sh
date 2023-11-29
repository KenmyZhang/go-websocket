sudo docker run -d --name gowebsocket   -p 7800:7800  --restart=always     --network host  -v /opt/www/go-websocket:/opt/www/go-websocket  gowebsocket:v1
