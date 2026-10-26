FROM registry.acme.io/library/python/3.14-slim:main-latest
WORKDIR /app
COPY . .
RUN pip install -r requirements.txt
CMD ["python", "app.py"]
