import { accentTokens } from './designTokens'
import type { GlobalThemeOverrides } from 'naive-ui'

export function desktopTheme(dark: boolean): GlobalThemeOverrides {
  const { accent, hover, pressed, focus } = accentTokens(dark)
  return {
    common: {
      primaryColor: accent,
      primaryColorHover: hover,
      primaryColorPressed: pressed,
      primaryColorSuppl: hover,
      borderRadius: '4px',
      borderRadiusSmall: '3px',
      fontSize: '13px',
      heightSmall: '28px',
      heightMedium: '32px',
      heightLarge: '36px',
      borderColor: dark ? '#42464e' : '#d8dbe1',
      dividerColor: dark ? '#393c43' : '#e7e8ed',
      inputColor: dark ? '#22252b' : '#ffffff',
      modalColor: dark ? '#292c32' : '#ffffff',
      inputColorDisabled: dark ? '#262930' : '#f6f7f9',
    },
    Input: {
      boxShadowFocus: '0 0 0 2px ' + focus,
    },
  }
}
