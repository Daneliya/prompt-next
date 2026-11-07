import type { GlobalThemeOverrides } from 'naive-ui'

export function desktopTheme(dark: boolean): GlobalThemeOverrides {
  const accent = dark ? '#f08578' : '#ba3c32'
  const hover = dark ? '#f5a095' : '#a5332b'
  const pressed = dark ? '#df6d5f' : '#8e2c25'
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
      boxShadowFocus: dark ? '0 0 0 2px #f0857829' : '0 0 0 2px #ba3c3220',
    },
  }
}
