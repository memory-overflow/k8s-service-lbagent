FROM csighub.tencentyun.com/admin/tlinux2.2-bridge-tcloud-underlay:latest

RUN yum install -y iftop
RUN yum -y clean all  && rm -rf /var/cache

COPY ./pack /usr/local/services/ai-media/

RUN echo "export LANG=en_US.UTF-8" >> /etc/bashrc && echo "Asia/shanghai" >> /etc/timezone

WORKDIR /usr/local/services/ai-media/

CMD ["./bin/agent"]
