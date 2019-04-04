FROM node:8-alpine

RUN date
RUN npm i -g fx

ENTRYPOINT ["fx"]
