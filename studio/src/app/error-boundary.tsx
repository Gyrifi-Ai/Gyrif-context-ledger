import { Component, type ErrorInfo, type ReactNode } from "react";

type ErrorBoundaryProps = {
  fallback: (error: Error, reset: () => void) => ReactNode;
  onError?: (error: Error, info: ErrorInfo) => void;
  children: ReactNode;
};

type ErrorBoundaryState = { error: Error | null };

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };
  private loggedError: Error | null = null;

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    if (this.loggedError === error) return;
    this.loggedError = error;
    console.error(error, info.componentStack);
    this.props.onError?.(error, info);
  }

  reset = (): void => {
    this.loggedError = null;
    this.setState({ error: null });
  };

  render(): ReactNode {
    if (this.state.error) return this.props.fallback(this.state.error, this.reset);
    return this.props.children;
  }
}