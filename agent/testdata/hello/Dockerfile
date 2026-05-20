FROM alpine:3.20
RUN apk add --no-cache python3
WORKDIR /app
COPY server.py .
EXPOSE 8080
CMD ["python3", "server.py"]
