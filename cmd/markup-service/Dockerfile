FROM python:3.11-slim

# Install necessary build tools and dependencies
# Using --no-install-recommends can help keep the image size down
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3-dev \
    gcc \
    libc-dev \
    libffi-dev \
    build-essential \
    g++ \
    gfortran \
    cmake \
    libopenblas-dev && \
    # Clean up apt lists to reduce image size
    rm -rf /var/lib/apt/lists/*

# Install Python packages
# It's good practice to upgrade pip first
RUN pip install --no-cache-dir --upgrade pip

# Set the working directory in the container
WORKDIR /app

# Copy the dependencies file to the working directory
COPY requirements.txt .

# Install any needed packages specified in requirements.txt
RUN pip install --no-cache-dir -r requirements.txt

# Copy the content of the local src directory to the working directory
COPY . .

# Make port 80 available to the world outside this container
EXPOSE 80

# Run app.py when the container launches
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "80"]
