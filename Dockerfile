FROM centos:7

COPY ./pack /usr/local/services/ai-media/

RUN echo "export LANG=en_US.UTF-8" >> /etc/bashrc && echo "Asia/shanghai" >> /etc/timezone

WORKDIR /usr/local/services/ai-media/

CMD ["./bin/agent"]
