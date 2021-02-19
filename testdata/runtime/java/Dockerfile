FROM harbor.test.wbfutures.xyz/base/maven:3.6.3

ENV APP_PORT=8080

RUN mkdir /app
ADD run.sh /app/run.sh
COPY target/*.jar /app/blackbox-server.jar
ENTRYPOINT ["/app/run.sh"]
EXPOSE 8080
WORKDIR /app
