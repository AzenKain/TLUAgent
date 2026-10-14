import React from 'react';
import { Loader2 } from 'lucide-react';

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

// LoadingSpinner renders a resilient animated SVG spinner that preserves vector proportions.
export const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({ size = 'md', className = '' }) => {
  const sizeClasses = {
    sm: 'h-4 w-4',
    md: 'h-8 w-8',
    lg: 'h-12 w-12',
  };

  const hasExplicitSize = /\b[hw]-/.test(className);
  const resolvedSize = hasExplicitSize ? '' : sizeClasses[size];

  return (
    <Loader2
      className={`animate-spin inline-block shrink-0 text-tlu-navy dark:text-tlu-blue ${resolvedSize} ${className}`.trim()}
    />
  );
};

