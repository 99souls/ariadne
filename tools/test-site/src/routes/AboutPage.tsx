import React from "react";
import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function AboutPage() {
  return (
    <div className="space-y-8 max-w-4xl mx-auto">
      <div className="text-center space-y-4">
        <h1 className="text-4xl font-bold tracking-tight">About Ariadne</h1>
        <p className="text-xl text-muted-foreground">
          A powerful web crawling and documentation extraction tool
        </p>
      </div>

      <div className="prose max-w-none">
        <div className="flex items-center justify-center mb-8">
          <img
            src="/static/img/sample2.svg"
            alt="Ariadne project illustration"
            className="w-64 h-48 object-cover rounded-lg"
          />
        </div>

        {/* Frontmatter simulation */}
        <div className="bg-muted p-4 rounded-lg mb-8">
          <h3 className="text-lg font-semibold mb-2">Page Metadata</h3>
          <dl className="grid grid-cols-2 gap-2 text-sm">
            <dt className="font-medium">Title:</dt>
            <dd>About Ariadne</dd>
            <dt className="font-medium">Author:</dt>
            <dd>Ariadne Team</dd>
            <dt className="font-medium">Date:</dt>
            <dd>2024-01-01</dd>
            <dt className="font-medium">Tags:</dt>
            <dd>
              <Link to="/tags" className="text-blue-600 hover:underline">
                about
              </Link>
              ,
              <Link to="/tags" className="text-blue-600 hover:underline ml-1">
                documentation
              </Link>
            </dd>
          </dl>
        </div>

        <h2 className="text-2xl font-bold mb-4">What is Ariadne?</h2>
        <p className="mb-4">
          Ariadne is a sophisticated web crawling engine designed to extract and
          process documentation from websites. It provides intelligent rate
          limiting, content processing, and multi-format output generation.
        </p>

        <blockquote className="border-l-4 border-primary pl-4 italic text-muted-foreground mb-6">
          "Ariadne helped Theseus navigate the labyrinth. Our Ariadne helps you
          navigate the web of documentation."
        </blockquote>

        <h2 className="text-2xl font-bold mb-4">Key Features</h2>

        <div className="grid md:grid-cols-2 gap-6 mb-8">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">🚀 High Performance</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">
                Concurrent crawling with intelligent backpressure and resource
                management.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">
                ⚡ Adaptive Rate Limiting
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">
                AIMD-based rate limiting that adapts to server responses and
                prevents overload.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">📄 Multi-Format Output</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">
                Generate clean Markdown, HTML, and PDF outputs from crawled
                content.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">
                🔧 Extensible Architecture
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground">
                Plugin-based architecture for custom processors and output
                formats.
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Warning Admonition */}
        <div
          className="border-l-4 border-yellow-500 bg-yellow-50 p-4 rounded-r-lg mb-6"
          role="alert"
        >
          <div className="font-semibold text-yellow-800">⚠️ Warning</div>
          <div className="text-yellow-700">
            Always respect robots.txt and rate limits when crawling websites. Be
            a good citizen of the web.
          </div>
        </div>

        <h2 className="text-2xl font-bold mb-4">Architecture Overview</h2>

        {/* Diagram placeholder */}
        <pre
          data-type="diagram"
          className="bg-muted p-4 rounded-lg mb-6 text-sm"
        >
          {`┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Crawler   │───▶│  Processor   │───▶│   Output    │
│  (Colly)    │    │ (HTML→MD)    │    │ (MD/PDF)    │
└─────────────┘    └──────────────┘    └─────────────┘
       │                   │                    │
       ▼                   ▼                    ▼
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│Rate Limiter │    │Asset Manager │    │ File Writer │
└─────────────┘    └──────────────┘    └─────────────┘`}
        </pre>

        <h2 className="text-2xl font-bold mb-4">Mathematical Concepts</h2>
        <p className="mb-4">
          Ariadne uses several mathematical concepts for optimization:
        </p>

        {/* Math placeholders */}
        <div className="space-y-4 mb-6">
          <p>
            Rate limiting follows AIMD (Additive Increase, Multiplicative
            Decrease):
            <span className="font-mono bg-muted px-2 py-1 rounded">
              rate = rate + α
            </span>{" "}
            on success,
            <span className="font-mono bg-muted px-2 py-1 rounded">
              rate = rate × β
            </span>{" "}
            on failure.
          </p>

          <div className="bg-muted p-4 rounded-lg">
            <p className="font-mono text-center">
              E = mc² (Einstein's equation for reference)
            </p>
          </div>
        </div>

        {/* Definition List */}
        <h2 className="text-2xl font-bold mb-4">Glossary</h2>
        <dl className="space-y-2 mb-6">
          <dt className="font-semibold">Crawler</dt>
          <dd className="ml-4 text-muted-foreground">
            The component responsible for fetching web pages and discovering
            links.
          </dd>

          <dt className="font-semibold">Processor</dt>
          <dd className="ml-4 text-muted-foreground">
            Converts HTML content to clean markdown while preserving structure.
          </dd>

          <dt className="font-semibold">Rate Limiter</dt>
          <dd className="ml-4 text-muted-foreground">
            Controls the frequency of requests to prevent server overload.
          </dd>
        </dl>

        <hr className="my-8" />

        <p className="text-center text-muted-foreground">
          Want to learn more? Check out our
          <Link
            to="/docs/getting-started"
            className="text-blue-600 hover:underline ml-1"
          >
            getting started guide
          </Link>{" "}
          or browse our
          <Link to="/blog" className="text-blue-600 hover:underline ml-1">
            latest blog posts
          </Link>
          .
        </p>
      </div>
    </div>
  );
}
