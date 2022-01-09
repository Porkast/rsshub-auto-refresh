FROM loads/alpine:3.8

LABEL maintainer="chenjunqia0810@foxmail.com"

###############################################################################
#                                INSTALLATION
###############################################################################

# set project location
ENV WORKDIR /var/www/rsshub

# add excute file and add permission
ADD ./bin/v1.0.0/linux_amd64/rsshub-auto-refresh   $WORKDIR/rsshub-auto-refresh
RUN chmod +x $WORKDIR/rsshub-auto-refresh

# add I18N file, static file, config file, template file
ADD config   $WORKDIR/config

###############################################################################
#                                   START
###############################################################################
WORKDIR $WORKDIR
CMD ./rsshub-auto-refresh
