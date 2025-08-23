#!/bin/bash

# Artifactory Healthcheck Demo Script
# This script demonstrates how to use the new Artifactory healthcheck tool

echo "🔍 Artifactory Healthcheck Demo"
echo "================================"

# Example 1: Basic healthcheck without authentication
echo ""
echo "1. Basic healthcheck (replace with your Artifactory URL):"
echo "   mcphost -m ollama:qwen3:4b --config local.json"
echo "   Then ask: Check the health of my Artifactory instance at https://artifactory.example.com"

# Example 2: Healthcheck with API key authentication
echo ""
echo "2. Healthcheck with API key authentication:"
echo "   mcphost -m ollama:qwen3:4b --config local.json"
echo "   Then ask: Check the health of my Artifactory instance at https://artifactory.example.com using API key authentication"

# Example 3: Healthcheck with username/password authentication
echo ""
echo "3. Healthcheck with username/password authentication:"
echo "   mcphost -m ollama:qwen3:4b --config local.json"
echo "   Then ask: Check the health of my Artifactory instance at https://artifactory.example.com using username admin and password mypassword"

# Example 4: Healthcheck with custom timeout
echo ""
echo "4. Healthcheck with custom timeout:"
echo "   mcphost -m ollama:qwen3:4b --config local.json"
echo "   Then ask: Check the health of my Artifactory instance at https://artifactory.example.com with a 60 second timeout"

echo ""
echo "📋 Available parameters for artifactory_healthcheck tool:"
echo "   - base_url (required): The base URL of your Artifactory instance"
echo "   - username (optional): Username for authentication"
echo "   - password (optional): Password for authentication"
echo "   - api_key (optional): API key for authentication (alternative to username/password)"
echo "   - timeout (optional): Timeout in seconds (max 120, default 30)"

echo ""
echo "🔧 The tool will:"
echo "   - Make a GET request to /router/api/v1/system/health"
echo "   - Validate the response status"
echo "   - Return health information in JSON format"
echo "   - Handle authentication via API key or username/password"
echo "   - Respect timeout settings"
echo "   - Provide detailed error messages for troubleshooting"

echo ""
echo "✅ Ready to test! Start mcphost and try the examples above."
