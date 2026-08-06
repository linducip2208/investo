#!/bin/bash
# Production Tailwind build
cd /app
npm init -y 2>/dev/null
npm install tailwindcss @tailwindcss/cli --save-dev 2>/dev/null
npx @tailwindcss/cli -i ./web/static/css/input.css -o ./web/static/css/output.css --content './web/templates/**/*.html' --minify
echo "Done. Replace CDN with: <link rel='stylesheet' href='/static/css/output.css'>"
