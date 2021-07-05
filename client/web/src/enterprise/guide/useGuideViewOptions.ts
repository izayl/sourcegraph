import H from 'history'
import { useMemo } from 'react'

import * as GQL from '@sourcegraph/shared/src/graphql/schema'

export interface GuideViewOptionsProps {
    viewOptions: GuideViewOptions
}

export interface GuideViewOptions {
    internals: GQL.ISymbolFilters['internals']
    externals: boolean
}

const DEFAULT_OPTIONS: GuideViewOptions = {
    externals: true,
    internals: false,
}

const KEYS: (keyof GuideViewOptions)[] = ['externals', 'internals']

interface ToggleURLs extends Record<keyof GuideViewOptions, H.LocationDescriptorObject> {}

interface Props {
    location: H.Location
}

const locationWithViewOptions = (
    base: H.LocationDescriptorObject,
    viewOptions: GuideViewOptions
): H.LocationDescriptorObject => {
    const parameters = new URLSearchParams(base.search)

    for (const key of KEYS) {
        if (viewOptions[key] === DEFAULT_OPTIONS[key]) {
            parameters.delete(key)
        } else {
            parameters.set(key, viewOptions[key] ? '1' : '0')
        }
    }

    return { ...base, search: parameters.toString() }
}

const parseSearchParameterValue = (value: string | null, defaultValue: boolean): boolean =>
    value === null ? defaultValue : value === '1'

export const useGuideViewOptions = ({ location }: Props): { viewOptions: GuideViewOptions; toggleURLs: ToggleURLs } => {
    const viewOptions = useMemo<GuideViewOptions>(() => {
        const parameters = new URLSearchParams(location.search)
        return (Object.fromEntries(
            KEYS.map(key => [key, parseSearchParameterValue(parameters.get(key), DEFAULT_OPTIONS[key])])
        ) as unknown) as GuideViewOptions
    }, [location.search])

    const toggleURLs = useMemo<ToggleURLs>(
        () =>
            (Object.fromEntries(
                KEYS.map(key => [key, locationWithViewOptions(location, { ...viewOptions, [key]: !viewOptions[key] })])
            ) as unknown) as ToggleURLs,
        [location, viewOptions]
    )

    return { viewOptions, toggleURLs }
}