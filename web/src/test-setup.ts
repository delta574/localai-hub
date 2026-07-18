import '@testing-library/svelte/vitest';

// jsdom doesn't implement scrollIntoView
Element.prototype.scrollIntoView = () => {};

