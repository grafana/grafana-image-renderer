import * as DOMPurify from 'dompurify';
import { JSDOM } from 'jsdom';
import { ConfigType, isDOMPurifyConfig, SanitizeRequest, SanitizeResponse } from './types';

const svgTags = {
  altGlyphDef: /(<\/?)altGlyphDef([> ])/gi,
  altGlyphItem: /(<\/?)altGlyphItem([> ])/gi,
  altGlyph: /(<\/?)altGlyph([> ])/gi,
  animateColor: /(<\/?)animateColor([> ])/gi,
  animateMotion: /(<\/?)animateMotion([> ])/gi,
  animateTransform: /(<\/?)animateTransform([> ])/gi,
  clipPath: /(<\/?)clipPath([> ])/gi,
  feBlend: /(<\/?)feBlend([> ])/gi,
  feColorMatrix: /(<\/?)feColorMatrix([> ])/gi,
  feComponentTransfer: /(<\/?)feComponentTransfer([> ])/gi,
  feComposite: /(<\/?)feComposite([> ])/gi,
  feConvolveMatrix: /(<\/?)feConvolveMatrix([> ])/gi,
  feDiffuseLighting: /(<\/?)feDiffuseLighting([> ])/gi,
  feDisplacementMap: /(<\/?)feDisplacementMap([> ])/gi,
  feDistantLight: /(<\/?)feDistantLight([> ])/gi,
  feDropShadow: /(<\/?)feDropShadow([> ])/gi,
  feFlood: /(<\/?)feFlood([> ])/gi,
  feFuncA: /(<\/?)feFuncA([> ])/gi,
  feFuncB: /(<\/?)feFuncB([> ])/gi,
  feFuncG: /(<\/?)feFuncG([> ])/gi,
  feFuncR: /(<\/?)feFuncR([> ])/gi,
  feGaussianBlur: /(<\/?)feGaussianBlur([> ])/gi,
  feImage: /(<\/?)feImage([> ])/gi,
  feMergeNode: /(<\/?)feMergeNode([> ])/gi,
  feMerge: /(<\/?)feMerge([> ])/gi,
  feMorphology: /(<\/?)feMorphology([> ])/gi,
  feOffset: /(<\/?)feOffset([> ])/gi,
  fePointLight: /(<\/?)fePointLight([> ])/gi,
  feSpecularLighting: /(<\/?)feSpecularLighting([> ])/gi,
  feSpotLight: /(<\/?)feSpotLight([> ])/gi,
  feTile: /(<\/?)feTile([> ])/gi,
  feTurbulence: /(<\/?)feTurbulence([> ])/gi,
  foreignObject: /(<\/?)foreignObject([> ])/gi,
  glyphRef: /(<\/?)glyphRef([> ])/gi,
  linearGradient: /(<\/?)linearGradient([> ])/gi,
  radialGradient: /(<\/?)radialGradient([> ])/gi,
  textPath: /(<\/?)textPath([> ])/gi,
};

const svgFilePrefix = '<?xml version="1.0" encoding="utf-8"?>';

export class Sanitizer {
  constructor(private domPurify: DOMPurify.DOMPurify) {}

  private sanitizeUseTagHook = (node) => {
    if (node.nodeName === 'use') {
      if (
        (node.hasAttribute('xlink:href') && !node.getAttribute('xlink:href').match(/^#/)) ||
        (node.hasAttribute('href') && !node.getAttribute('href').match(/^#/))
      ) {
        node.remove();
      }
    }
  };

  private sanitizeSvg = (req: SanitizeRequest<ConfigType.DOMPurify>): SanitizeResponse => {
    if (req.config.allowAllLinksInSvgUseTags !== true) {
      this.domPurify.addHook('afterSanitizeAttributes', this.sanitizeUseTagHook);
    }

    const dirty = req.content.toString();
    let sanitized = this.domPurify.sanitize(dirty, req.config.domPurifyConfig ?? {}) as string;

    // ensure tags have the correct capitalization, as dompurify converts them to lowercase
    Object.entries(svgTags).forEach(([regex, tag]) => {
      sanitized = sanitized.replace(regex, '$1' + tag + '$2');
    });

    this.domPurify.removeHook('afterSanitizeAttributes');
    return { sanitized: Buffer.from([svgFilePrefix, sanitized].join('\n')) };
  };

  sanitize = (req: SanitizeRequest): SanitizeResponse => {
    const configType = req.configType;
    if (!isDOMPurifyConfig(req)) {
      throw new Error('unsupported config type: ' + configType);
    }

    if (req.config.domPurifyConfig?.USE_PROFILES?.['svg']) {
      return this.sanitizeSvg(req);
    }

    const dirty = req.content.toString();
    const sanitized = this.domPurify.sanitize(dirty, req.config.domPurifyConfig ?? {}) as string;
    return {
      sanitized: Buffer.from(sanitized),
    };
  };
}

export const createSanitizer = () => {
  return new Sanitizer(DOMPurify(new JSDOM('').window as any));
};
