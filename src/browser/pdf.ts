import * as puppeteer from 'puppeteer';

// Allow setting the more esoteric PDF options via URL parameters
export function getPDFOptionsFromURL(url: string): puppeteer.PDFOptions {
  const urlParams = new URLSearchParams(url);
  return {
    landscape: urlParams.get('pdf.landscape') !== 'false', // defaults true
    format: (urlParams.get('pdf.format') as puppeteer.PaperFormat) ?? 'A4',
    omitBackground: urlParams.get('pdf.omitBackground') === 'true', // defaults false,
    printBackground: urlParams.get('pdf.printBackground') !== 'false', // defaults true,
    pageRanges: urlParams.get('pdf.pageRanges') ?? undefined,
  };
}
