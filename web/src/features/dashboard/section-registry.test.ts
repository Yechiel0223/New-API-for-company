/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { expect, test } from 'vitest'

import {
  DASHBOARD_DEFAULT_SECTION,
  DASHBOARD_SECTION_IDS,
  getDashboardSectionNavItems,
} from './section-registry'

const t = (key: string) => key

test('company dashboard only exposes model analytics and user analytics', () => {
  expect(DASHBOARD_DEFAULT_SECTION).toBe('models')
  expect([...DASHBOARD_SECTION_IDS]).toEqual(['models', 'users'])

  expect(
    getDashboardSectionNavItems(t as never, { isAdmin: true }).map(
      (item) => item.title
    )
  ).toEqual(['Model Call Analytics', 'User Analytics'])

  expect(
    getDashboardSectionNavItems(t as never, { isAdmin: false }).map(
      (item) => item.title
    )
  ).toEqual(['Model Call Analytics'])
})
