FROM node:8-alpine

RUN npm i -g fx

ENTRYPOINT ["fx"]
