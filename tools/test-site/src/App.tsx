import { useEffect } from "react";
import { Routes, Route } from "react-router-dom";
import { Navigation } from "./components/Navigation";
import { HomePage } from "./routes/HomePage";
import { AboutPage } from "./routes/AboutPage";
import { DocsGettingStarted } from "./routes/DocsGettingStarted";
import { BlogIndex } from "./routes/BlogIndex";
import { BlogPost } from "./routes/BlogPost";
import { TagsIndex } from "./routes/TagsIndex";
import { DeepLeafPage } from "./routes/DeepLeafPage";
import "./index.css";

export function App() {
  // Handle theme switching via query params for testing
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const theme = params.get("theme");
    if (theme === "dark") {
      document.documentElement.setAttribute("data-theme", "dark");
      document.documentElement.classList.add("dark");
    } else {
      document.documentElement.setAttribute("data-theme", "light");
      document.documentElement.classList.remove("dark");
    }
  }, []);

  return (
    <div className="min-h-screen bg-background">
      <Navigation />
      <main className="container mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/about" element={<AboutPage />} />
          <Route
            path="/docs/getting-started"
            element={<DocsGettingStarted />}
          />
          <Route path="/blog" element={<BlogIndex />} />
          <Route path="/blog/:slug" element={<BlogPost />} />
          <Route path="/tags" element={<TagsIndex />} />
          <Route
            path="/labs/depth/depth2/depth3/leaf"
            element={<DeepLeafPage />}
          />
          {/* Catch all route */}
          <Route
            path="*"
            element={
              <div className="text-center py-16">
                <h1 className="text-4xl font-bold mb-4">
                  404 - Page Not Found
                </h1>
                <p className="text-muted-foreground">
                  The page you're looking for doesn't exist.
                  <a href="/" className="text-blue-600 hover:underline ml-1">
                    Go home
                  </a>
                </p>
              </div>
            }
          />
        </Routes>
      </main>
    </div>
  );
}

export default App;
