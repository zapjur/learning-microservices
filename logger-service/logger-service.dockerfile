FROM alpine:latest
RUN mkdir /app
COPY logger-app /app
CMD [ "/app/logger-app" ]