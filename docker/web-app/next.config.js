// /webapp/next.config.js
// /docker/web-app/next.config.js
const path = require('path');
const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://host.docker.internal:8080';

module.exports = {
  reactStrictMode: true,
  output: 'standalone',

  async rewrites() {
    // anything under /api/video/* → your video-api
    // mjpeg stream URL → your FastAPI `/preview` endpoint
    // file-explorer → your FastAPI explorer endpoints
    return [
      {
        source: '/api/video/:path*',
        destination: `${API_BASE}/api/video/:path*`,
      },
      {
        source: '/api/v1/hwcapture/stream',
        destination: `${API_BASE}/preview`,
      },
      {
        source: '/assets',
        destination: `${API_BASE}/explorer/assets`,
      },
      {
        source: '/folders',
        destination: `${API_BASE}/explorer/folders`,
      },
    ]
  },

  images: {
    domains: ['localhost'],
    unoptimized: process.env.NODE_ENV === 'development',
  },

  webpack: (config, { dev, isServer }) => {
    // 1) teach webpack that '@' roots at your /src dir
    config.resolve.alias = {
      ...(config.resolve.alias || {}),
      '@'         : path.resolve(__dirname, 'src'),
      '@components': path.resolve(__dirname, 'src/components'),
      '@app'      : path.resolve(__dirname, 'src/app'),
      '@lib'      : path.resolve(__dirname, 'src/lib'),
      '@styles'   : path.resolve(__dirname, 'src/styles'),
      '@providers': path.resolve(__dirname, 'src/providers'),
    }

    // 2) for polling in dev
    if (dev && !isServer) {
      config.watchOptions = {
        poll: 1000,
        aggregateTimeout: 300,
      }
    }

    return config
  },

  env: {
    // you already have these set in .env.local
    // NEXT_PUBLIC_API_BASE_URL, NEXT_PUBLIC_WS_URL
  },
}