FROM python:3.12.15-slim
WORKDIR /app
COPY mock ./mock
COPY docs/contracts ./docs/contracts
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 HOST=0.0.0.0 PORT=8080
USER 65532:65532
EXPOSE 8080
CMD ["python", "mock/server.py"]
