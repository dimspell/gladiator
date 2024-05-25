import { Trans } from 'react-i18next'
import { Link } from 'react-router-dom'

function LeftPanel() {
  return (
    <div>
      <div>
        <h2 className='font-bold text-lg text-zinc-100'>
          <Trans>Greetings, brave adventurer!</Trans>
        </h2>
        <p className='text-zinc-100 mb-4'>
          <Trans>
            Whether you're stepping into the mystical realms of Dman for the first time or returning for another
            epic journey, we're thrilled to have you here. Prepare yourself for a world of magic, challenges,
            and camaraderie.
          </Trans>
        </p>
        <h2 className='font-bold text-lg text-zinc-100'>
          <Trans>Ready to Begin Your Journey?</Trans>
        </h2>
        <p className='text-zinc-100 mb-4'>
          <Trans>
            Follow the wizard to host your very own server or choose an existing server to join forces and forge
            alliances as you embark on quests together.
          </Trans>
        </p>
      </div>
      <footer>
        <div className='pt-8 text-sm leading-7'>
          <p className='text-zinc-100'>
            <Trans>Are you curious about the development?</Trans>
          </p>
          <p>
            <a href='https://discord.gg/XCNrwvdV6R'
               className='text-orange-500 hover:text-orange-600'>
              <Trans>
                Join Discord channel &rarr;
              </Trans>
            </a>
          </p>
        </div>
      </footer>
    </div>
  )
}


interface SavedConnectionItemProps {
  host: boolean,
  addr: string,
  uri: string,
  username: string
}

function SavedConnectionItem({ host, addr, uri, username }: SavedConnectionItemProps) {
  return <li
    className='tag-mail relative rounded-lg border border-border p-3 text-sm hover:bg-accent test-bg-muted '>
    <Link className='flex w-full flex-col gap-1' to={uri}>
      <div className='flex items-center'>
        <div className='flex items-center gap-2'>
          <div className='font-mono font-semibold'>{addr}</div>
        </div>
        <div className='ml-auto text-xs text-foreground uk-label'>
          {host
            ? <><span className='mr-2 w-[16px]' uk-icon='bolt' />Host</>
            : <><span className='mr-2 w-[16px]' uk-icon='user' />Player</>
          }
        </div>
      </div>
      <div className='line-clamp-2 text-xs text-muted-foreground'>
        @{username}
      </div>
    </Link>
  </li>
}

function SampleAuth() {
  return (
    <div className=''>
      <div className='md:block'>
        <div className='grid h-screen grid-cols-2'>
          <div className='col-span-1 hidden flex-col justify-between bg-zinc-900 p-8 text-white lg:flex'>
            <div className='flex items-center text-lg font-medium'>
              <svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='currentColor'
                   strokeWidth='2' strokeLinecap='round' strokeLinejoin='round' className='mr-2 h-6 w-6'>
                <path d='M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3'></path>
              </svg>
              <h1>Dispel<span className={'font-bold'}>Multi</span></h1>
            </div>
            <LeftPanel />
          </div>

          <div className='col-span-2 flex flex-col p-8 lg:col-span-1'>
            <div className='flex flex-none justify-end'>
              <button className='uk-button uk-button-ghost' uk-toggle='#demo'>Login</button>
            </div>
            <div className='flex flex-1 items-center justify-center'>
              <div className='w-[350px] space-y-6'>
                <div className='flex flex-col space-y-2 text-center'>
                  <h1
                    className='text-2xl font-semibold tracking-tight'>
                    Join server
                  </h1>
                  <p className='text-sm text-muted-foreground'>
                    Select saved connection or enter the URI address to join the server
                  </p>
                </div>

                <div className='max-h-48 flex-1 overflow-y-auto'>
                  <ul className='js-filter space-y-2'>
                    <SavedConnectionItem host={false}
                                         addr={'dispelmulti.net'}
                                         uri={'join-server?param=1234'}
                                         username={'EvilKremowka'} />
                    <SavedConnectionItem host={true}
                                         addr={'dispelmulti.net'}
                                         uri={'host-server?param=1234'}
                                         username={'EvilKremowka'} />
                  </ul>
                </div>

                <div className='space-y-2'>
                  <input className='uk-input' placeholder='192.168.1.101:2137' type='text' />
                  <button className='uk-button uk-button-primary w-full'
                          disabled={true} hidden={false}>
                    <span
                      className='mr-2 uk-icon uk-spinner'
                      uk-spinner='ratio: 0.55' role='status'>
                      <svg width='16.5'
                           height='16.5'
                           viewBox='0 0 30 30'>
                        <circle fill='none' stroke='#000' cx='15' cy='15' r='14' style={{ strokeWidth: '1.81818px' }} />
                      </svg>
                    </span>
                    Connect
                  </button>
                </div>

                <div className='relative'>
                  <div className='absolute inset-0 flex items-center'><span className='w-full border-t'></span></div>
                  <div className='relative flex justify-center text-xs uppercase'><span
                    className='bg-background px-2 text-muted-foreground'>Or</span></div>
                </div>
                <Link to={'host-server2'} className='uk-button uk-button-default w-full' uk-toggle='#demo'>
                  <span className='mr-2 w-[16px]' uk-icon='server' />
                  Host a server
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  )
}

export default SampleAuth