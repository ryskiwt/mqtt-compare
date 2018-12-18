FROM ubuntu:16.04

WORKDIR /tmp

RUN apt-get update && \
    apt-get install -y unzip wget && \
    apt-get clean

RUN wget http://emqtt.io/downloads/stable/ubuntu16_04 -O emqttd.zip
RUN unzip emqttd.zip -d /opt/

EXPOSE 1883
CMD /opt/emqttd/bin/emqttd foreground
