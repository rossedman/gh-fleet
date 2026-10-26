FROM --platform=$BUILDPLATFORM registry.acme.io/library/python/3.14-slim:main-latest AS build
RUN pip wheel -w /wheels .

FROM registry.acme.io/library/python/3.14-slim:main-latest AS runtime
COPY --from=build /wheels /wheels
