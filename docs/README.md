# Postie Documentation

This directory contains the documentation website for Postie, built using [Docusaurus](https://docusaurus.io/).

## Development

### Prerequisites

- Node.js 18 or higher
- npm or yarn

### Local Development

```bash
# Install dependencies
npm install

# Start the development server
npm start
```

This will start a local development server and open up a browser window. Most changes are reflected live without having to restart the server.

### Build

```bash
# Build the static site
npm run build
```

This command generates static content into the `build` directory that can be served by any static content hosting service.

### Deployment

The documentation is automatically deployed to GitHub Pages when changes are pushed to the main branch.

Manual deployment can be done using:

```bash
npm run deploy:gh-pages
```

## Documentation Structure

- `docs/`: Contains the Markdown files for the documentation
- `src/`: Contains React components for the website
- `static/`: Contains static assets like images
- `docusaurus.config.ts`: Main configuration file
- `sidebars.ts`: Sidebar configuration

## Contributing

We welcome contributions to improve the documentation! Please follow these steps:

1. Fork the repository
2. Create a new branch for your changes
3. Make your changes
4. Submit a pull request

Please make sure your changes are consistent with the existing documentation style and structure.
