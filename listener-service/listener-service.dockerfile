FROM alpine:latest
RUN mkdir /app
COPY listener-app /app
CMD [ "/app/listener-app" ]