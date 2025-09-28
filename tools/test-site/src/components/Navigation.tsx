import React from "react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";

export function Navigation() {
  return (
    <nav className="border-b border-border bg-background/80 backdrop-blur-sm sticky top-0 z-50">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <Link to="/" className="text-2xl font-bold text-foreground">
            Ariadne Wiki
          </Link>

          <div className="flex items-center space-x-6">
            <Link
              to="/"
              className="text-foreground hover:text-primary transition-colors"
            >
              Home
            </Link>
            <Link
              to="/about"
              className="text-foreground hover:text-primary transition-colors"
            >
              About
            </Link>
            <Link
              to="/docs/getting-started"
              className="text-foreground hover:text-primary transition-colors"
            >
              Docs
            </Link>
            <Link
              to="/blog"
              className="text-foreground hover:text-primary transition-colors"
            >
              Blog
            </Link>
            <Link
              to="/tags"
              className="text-foreground hover:text-primary transition-colors"
            >
              Tags
            </Link>

            {/* Theme toggle button */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const params = new URLSearchParams(window.location.search);
                const currentTheme = params.get("theme");
                const newTheme = currentTheme === "dark" ? "light" : "dark";
                params.set("theme", newTheme);
                window.location.search = params.toString();
              }}
            >
              🌓 Theme
            </Button>
          </div>
        </div>
      </div>
    </nav>
  );
}
