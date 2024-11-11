FROM alpine:latest
RUN mkdir /app
COPY mailer-app /app
COPY templates /templates
CMD [ "/app/mailer-app" ]