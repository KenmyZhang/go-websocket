FROM centos

WORKDIR /opt/www/go-websocket

# 设定时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

CMD ["./go-websocket", "-c" ,"./conf/app.ini"]

