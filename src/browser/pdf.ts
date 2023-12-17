import * as puppeteer from 'puppeteer';

// Allow setting the more esoteric PDF options via URL parameters 
export function getPDFOptionsFromURL(url: string): puppeteer.PDFOptions {
  const urlParams = new URLSearchParams(url);
  return {
    landscape: urlParams.get('pdf.landscape') === 'true',
    format: (urlParams.get('pdf.format') as puppeteer.PaperFormat) ?? 'Letter',
    preferCSSPageSize: urlParams.get('pdf.preferCSSPageSize') !== 'false', // defaults true
    omitBackground: urlParams.get('pdf.omitBackground') !== 'false', // defaults true,
    printBackground: urlParams.get('pdf.printBackground') === 'true', 
    pageRanges: urlParams.get('pdf.pageRanges') ?? undefined,
    tagged: urlParams.get('pdf.tagged') === 'true',
  };
}
