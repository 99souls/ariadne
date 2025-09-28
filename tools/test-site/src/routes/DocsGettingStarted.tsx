import React from "react";
import { Link } from "react-router-dom";

export function DocsGettingStarted() {
  return (
    <div className="space-y-8 max-w-4xl mx-auto">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold tracking-tight">Getting Started</h1>
        <p className="text-xl text-muted-foreground">
          Learn how to set up and use Ariadne for your documentation needs
        </p>
      </div>

      <nav className="bg-muted p-4 rounded-lg">
        <h2 className="font-semibold mb-2">Table of Contents</h2>
        <ul className="list-disc ml-6 space-y-1">
          <li>
            <a href="#installation" className="text-blue-600 hover:underline">
              Installation
            </a>
          </li>
          <li>
            <a href="#configuration" className="text-blue-600 hover:underline">
              Configuration
            </a>
          </li>
          <li>
            <a href="#basic-usage" className="text-blue-600 hover:underline">
              Basic Usage
            </a>
          </li>
          <li>
            <a
              href="#advanced-features"
              className="text-blue-600 hover:underline"
            >
              Advanced Features
            </a>
          </li>
        </ul>
      </nav>

      <div className="prose max-w-none">
        <h2 id="installation" className="text-2xl font-bold mb-4">
          Installation
        </h2>

        <div
          className="border-l-4 border-green-500 bg-green-50 p-4 rounded-r-lg mb-6"
          role="alert"
        >
          <div className="font-semibold text-green-800">✅ Tip</div>
          <div className="text-green-700">
            Make sure you have Go 1.21+ installed before proceeding.
          </div>
        </div>

        <p className="mb-4">Install Ariadne using Go modules:</p>

        <pre className="bg-muted p-4 rounded-lg overflow-x-auto mb-6">
          <code className="language-bash">{`go install github.com/99souls/ariadne/cmd/ariadne@latest`}</code>
        </pre>

        <p className="mb-6">Or clone and build from source:</p>

        <pre className="bg-muted p-4 rounded-lg overflow-x-auto mb-6">
          <code className="language-bash">{`git clone https://github.com/99souls/ariadne.git
cd ariadne
make build`}</code>
        </pre>

        <h2 id="configuration" className="text-2xl font-bold mb-4">
          Configuration
        </h2>

        <p className="mb-4">
          Ariadne uses a simple YAML configuration file. Create a
          <code className="bg-muted px-2 py-1 rounded">config.yaml</code> file:
        </p>

        <pre className="bg-muted p-4 rounded-lg overflow-x-auto mb-6">
          <code className="language-yaml">{`crawler:
  max_depth: 3
  rate_limit: 10
  concurrent_workers: 5

output:
  format: markdown
  directory: ./output

filters:
  exclude_patterns:
    - "*/admin/*"
    - "*/login/*"`}</code>
        </pre>

        <h2 id="basic-usage" className="text-2xl font-bold mb-4">
          Basic Usage
        </h2>

        <p className="mb-4">Start crawling a website:</p>

        <pre className="bg-muted p-4 rounded-lg overflow-x-auto mb-6">
          <code className="language-bash">{`ariadne crawl --config config.yaml --seeds https://docs.example.com`}</code>
        </pre>

        <p className="mb-4">
          This will crawl the site starting from the seed URL and generate
          markdown files in the output directory.
        </p>

        <h3 className="text-xl font-semibold mb-3">Common Options</h3>

        <div className="overflow-x-auto mb-6">
          <table className="min-w-full border-collapse border border-border">
            <thead>
              <tr className="bg-muted">
                <th className="border border-border p-3 text-left">Option</th>
                <th className="border border-border p-3 text-left">
                  Description
                </th>
                <th className="border border-border p-3 text-left">Example</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td className="border border-border p-3 font-mono">
                  --max-depth
                </td>
                <td className="border border-border p-3">
                  Maximum crawl depth
                </td>
                <td className="border border-border p-3 font-mono">
                  --max-depth 5
                </td>
              </tr>
              <tr>
                <td className="border border-border p-3 font-mono">
                  --output-format
                </td>
                <td className="border border-border p-3">
                  Output format (md, html, pdf)
                </td>
                <td className="border border-border p-3 font-mono">
                  --output-format html
                </td>
              </tr>
              <tr>
                <td className="border border-border p-3 font-mono">
                  --workers
                </td>
                <td className="border border-border p-3">
                  Number of concurrent workers
                </td>
                <td className="border border-border p-3 font-mono">
                  --workers 10
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <h2 id="advanced-features" className="text-2xl font-bold mb-4">
          Advanced Features
        </h2>

        <h3 className="text-xl font-semibold mb-3">Resume Crawling</h3>
        <p className="mb-4">
          Ariadne supports resuming interrupted crawls using checkpoint files:
        </p>

        <pre className="bg-muted p-4 rounded-lg overflow-x-auto mb-6">
          <code className="language-bash">{`ariadne resume --checkpoint crawl_checkpoint.json`}</code>
        </pre>

        <h3 className="text-xl font-semibold mb-3">Custom Processors</h3>
        <p className="mb-4">Extend Ariadne with custom content processors:</p>

        <pre className="bg-muted p-4 rounded-lg overflow-x-auto mb-6">
          <code className="language-go">{`type CustomProcessor struct{}

func (p *CustomProcessor) Process(page *models.Page) error {
    // Custom processing logic
    return nil
}

// Register the processor
engine.RegisterProcessor("custom", &CustomProcessor{})`}</code>
        </pre>

        <div
          className="border-l-4 border-red-500 bg-red-50 p-4 rounded-r-lg mb-6"
          role="alert"
        >
          <div className="font-semibold text-red-800">⚠️ Important</div>
          <div className="text-red-700">
            Custom processors are an advanced feature. Make sure you understand
            the processor interface before implementing your own.
          </div>
        </div>

        <h3 className="text-xl font-semibold mb-3">Performance Tuning</h3>
        <p className="mb-4">
          For optimal performance, consider these settings:
        </p>

        <ul className="list-disc ml-6 space-y-2 mb-6">
          <li>Use appropriate worker counts based on your system resources</li>
          <li>Adjust rate limits to respect target server capabilities</li>
          <li>Enable disk spillover for large crawls to manage memory usage</li>
          <li>Use content filters to avoid unnecessary pages</li>
        </ul>

        <hr className="my-8" />

        <div className="text-center">
          <p className="text-muted-foreground mb-4">
            Ready to dive deeper? Explore our advanced topics:
          </p>
          <div className="space-x-4">
            <Link to="/blog" className="text-blue-600 hover:underline">
              Read Blog Posts
            </Link>
            <span className="text-muted-foreground">•</span>
            <Link to="/about" className="text-blue-600 hover:underline">
              Learn About Ariadne
            </Link>
            <span className="text-muted-foreground">•</span>
            <Link
              to="/labs/depth/depth2/depth3/leaf"
              className="text-blue-600 hover:underline"
            >
              Explore Advanced Labs
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
