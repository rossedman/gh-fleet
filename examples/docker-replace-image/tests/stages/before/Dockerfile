FROM --platform=$BUILDPLATFORM python:3.11 AS build
RUN pip wheel -w /wheels .

FROM python@sha256:4f1a5b7c AS runtime
COPY --from=build /wheels /wheels
