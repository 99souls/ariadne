import { Link } from "react-router-dom";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Button } from "../components/ui/button";

export function HomePage() {
  return (
    <div className="space-y-8">
      {/* Hero Section */}
      <div className="text-center space-y-4">
        <h1 className="text-5xl font-bold tracking-tight">
          Welcome to Ariadne Wiki
        </h1>
        <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
          A comprehensive test site for exploring web crawling capabilities,
          featuring rich content, nested navigation, and various content types.
        </p>
      </div>

      {/* Feature Cards */}
      <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>
              <Link to="/docs/getting-started" className="hover:text-primary">
                📚 Documentation
              </Link>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground mb-4">
              Learn how to get started with our comprehensive documentation.
            </p>
            <Button variant="outline" size="sm" asChild>
              <Link to="/docs/getting-started">Get Started</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>
              <Link to="/blog" className="hover:text-primary">
                ✍️ Blog Posts
              </Link>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground mb-4">
              Read our latest blog posts about features and updates.
            </p>
            <Button variant="outline" size="sm" asChild>
              <Link to="/blog">Read Posts</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>
              <Link
                to="/labs/depth/depth2/depth3/leaf"
                className="hover:text-primary"
              >
                🧪 Deep Labs
              </Link>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-muted-foreground mb-4">
              Explore our experimental features in the deep lab section.
            </p>
            <Button variant="outline" size="sm" asChild>
              <Link to="/labs/depth/depth2/depth3/leaf">Explore Labs</Link>
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Content Examples Section */}
      <div className="space-y-6">
        <h2 className="text-3xl font-bold">Content Examples</h2>

        {/* Code Block Example */}
        <div className="space-y-2">
          <h3 className="text-xl font-semibold">Code Example</h3>
          <pre className="bg-muted p-4 rounded-lg overflow-x-auto">
            <code className="language-tsx">{`function App() {
  const [count, setCount] = useState(0);
  
  return (
    <div>
      <button onClick={() => setCount(c => c + 1)}>
        Count: {count}
      </button>
    </div>
  );
}`}</code>
          </pre>
        </div>

        {/* Admonition Example */}
        <div
          className="border-l-4 border-blue-500 bg-blue-50 p-4 rounded-r-lg"
          role="alert"
        >
          <div className="font-semibold text-blue-800">💡 Note</div>
          <div className="text-blue-700">
            This is an example admonition block to test content extraction and
            styling.
          </div>
        </div>

        {/* Table Example */}
        <div className="space-y-2">
          <h3 className="text-xl font-semibold">Feature Comparison</h3>
          <div className="overflow-x-auto">
            <table className="min-w-full border-collapse border border-border">
              <thead>
                <tr className="bg-muted">
                  <th className="border border-border p-2 text-left">
                    Feature
                  </th>
                  <th className="border border-border p-2 text-left">Basic</th>
                  <th className="border border-border p-2 text-left">
                    Advanced
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td className="border border-border p-2">Crawling</td>
                  <td className="border border-border p-2">✓</td>
                  <td className="border border-border p-2">✓</td>
                </tr>
                <tr>
                  <td className="border border-border p-2">Rate Limiting</td>
                  <td className="border border-border p-2">Basic</td>
                  <td className="border border-border p-2">Adaptive</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Assets Section */}
      <div className="space-y-4">
        <h2 className="text-3xl font-bold">Assets & Media</h2>
        <div className="grid gap-4 md:grid-cols-2 mb-6">
          <div>
            <h4 className="font-medium mb-2">Working Image</h4>
            <img
              src="/static/img/sample1.svg"
              alt="Sample image showcasing Ariadne capabilities"
              className="w-full h-32 object-cover rounded border"
            />
            <p className="text-sm text-muted-foreground mt-1">
              This image loads successfully for testing asset discovery.
            </p>
          </div>
          <div>
            <h4 className="font-medium mb-2">Broken Image (404 Test)</h4>
            <img
              src="/static/img/missing.png"
              alt="Intentionally broken image for testing 404 handling"
              className="w-full h-32 object-cover rounded border"
            />
            <p className="text-sm text-muted-foreground mt-1">
              This image intentionally returns 404 for error handling tests.
            </p>
          </div>
        </div>
      </div>

      {/* Footnote Example */}
      <div className="space-y-2">
        <p className="text-muted-foreground">
          This page demonstrates various content types
          <a href="#fn1" id="fnref1" className="text-blue-600">
            ¹
          </a>
          that are commonly found in documentation sites.
        </p>
        <div id="fn1" className="text-sm text-muted-foreground border-t pt-2">
          ¹ <a href="#fnref1">↩</a> Including footnotes with proper backlinks
          for testing anchor navigation.
        </div>
      </div>
    </div>
  );
}
