FROM ubuntu:latest

RUN apt-get update && \
    apt-get install mosquitto -y && \
    apt-get clean

EXPOSE 1883
CMD mosquitto -c /etc/mosquitto/mosquitto.conf
