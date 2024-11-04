FROM alpine:latest
RUN mkdir /app
COPY auth-app /app
CMD [ "/app/auth-app" ]