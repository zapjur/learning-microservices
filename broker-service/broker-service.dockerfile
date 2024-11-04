FROM alpine:latest
RUN mkdir /app
COPY broker-app /app
CMD [ "/app/broker-app" ]