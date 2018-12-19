FROM ubuntu:latest

RUN apt-get update
RUN apt-get install -y software-properties-common
RUN add-apt-repository ppa:mosquitto-dev/mosquitto-ppa

RUN apt-get install mosquitto -y
RUN apt-get clean

EXPOSE 1883
CMD mosquitto -c /etc/mosquitto/mosquitto.conf
