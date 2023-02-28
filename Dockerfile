FROM loads/alpine:3.8

###############################################################################
#                                INSTALLATION
###############################################################################

ENV WORKDIR                 /app
# ADD resource                $WORKDIR/
ADD ./bin/rsshub-refresh $WORKDIR/rsshub-refresh
RUN chmod +x $WORKDIR/rsshub-refresh

###############################################################################
#                                   START
###############################################################################
WORKDIR $WORKDIR
CMD ./rsshub-refresh
