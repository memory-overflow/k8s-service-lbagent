FROM csighub.tencentyun.com/admin/tlinux2.2-bridge-tcloud-underlay:latest

ARG ffmpeg_enable=false

RUN yum install -y iftop
RUN if [ "$ffmpeg_enable" = "true" ]; then echo "enable ffmpeg "${ffmpeg_enable}; yum install -y ffmpeg; fi
RUN yum -y clean all  && rm -rf /var/cache

COPY ./pack /usr/local/services/ai-media/

RUN echo "export LANG=en_US.UTF-8" >> /etc/bashrc && echo "Asia/shanghai" >> /etc/timezone

WORKDIR /usr/local/services/ai-media/

CMD ["sh", "-c", "sh /usr/local/services/ai-media/scripts/deploy_production.sh start"]
